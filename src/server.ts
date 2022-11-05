import { createServer, IncomingMessage, ServerResponse } from "http";
import { watch } from "chokidar";
import { WebSocketServer, WebSocket } from "ws";
import { build, Site } from "./builder";
import { debounce, time } from "./helpers";
import { join } from "path";
import { stat } from "fs/promises";
import { createReadStream } from "fs";
import { lookup } from "mrmime";
import { bold } from "picocolors";
import { reportError } from "./errors";

type Request = IncomingMessage;
type Response = ServerResponse<IncomingMessage>;

async function serveStaticFile(dir: string, req: Request, res: Response) {
  let file = join(dir, req.url || "/");

  try {
    let stats = await stat(file);

    if (stats.isDirectory()) {
      file = join(file, "index.html");
    }

    await stat(file);

    res.writeHead(200, "ok", {
      "Content-Type": lookup(file) || "text/plain",
    });

    createReadStream(file).pipe(res);
    return true;
  } catch {
    return false;
  }
}

async function tryStaticFile(dirs: string[], req: Request, res: Response) {
  for (let dir of dirs) {
    let served = await serveStaticFile(dir, req, res);
    if (served) return;
  }

  res.statusCode = 404;
  res.statusMessage = "Not found";
  res.end();
}

function createLiveReloadServer(dirs: string[]) {
  let server = createServer(async (req, res) => {
    return tryStaticFile(dirs, req, res);
  });

  let wss = new WebSocketServer({ server });
  let sockets = new Set<WebSocket>();
  wss.on("connection", socket => sockets.add(socket));

  function refresh() {
    for (let socket of sockets) {
      socket.send("reload");
    }
  }

  return {
    refresh,
    server,
  };
}

/**
 * @internal
 */
export async function serve(site: Site) {
  let { server, refresh } = createLiveReloadServer([site.outDir, site.publicDir]);

  let watcher = watch(site.rootDir, {
    ignored: /(_site|\.git|\.cache|dist|node_modules)/,
    cwd: site.rootDir,
    ignoreInitial: true,
  });

  async function _rebuild() {
    try {
      await time("built in", build(site));
      console.log();
      refresh();
    } catch (err: any) {
      reportError(err);
    }
  }

  // Rebuild needs to be debounced so that multiple simultaneous changes on
  // disk (deleting a dir, pasting 3 files etc) only trigger a single rebuild.
  let rebuild = debounce(_rebuild, 100);

  let eventNames = {
    "add": "added",
    "addDir": "added",
    "change": "changed",
    "unlink": "removed",
    "unlinkDir": "removed",
  };

  watcher.on("all", async (event, path) => {
    console.log(`${eventNames[event]}: ${path}`);
    rebuild();
  });

  // Use the non-debounced _rebuild to ensure that we've done a complete build
  // before starting the server.
  await _rebuild();

  let port = 3000;
  server.listen(port);
  console.log(`server listening on ${bold(`http://localhost:${port}`)}`);
}
