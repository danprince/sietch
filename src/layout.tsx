import { ComponentProps, Context, createContext, FunctionComponent } from "preact";
import { render as renderToString } from "preact-render-to-string";
import { useContext } from "preact/hooks";
import { Page, Site } from "./builder";

interface LayoutContext {
  site: Site;
  page: Page;
}

// We store the layout context against a symbol to guard against collisions.
let contextSymbol = Symbol.for("sietch-context");

function getLayoutContext(): Context<LayoutContext> {
  // We need to make sure this context only gets created once, as there are
  // multiple modules that might need to use it (e.g. the CLI and the version
  // that the user's layout imports). It's a bit of a nasty hack, but it saves
  // a lot of modules pain if we instead decide not to bundle.

  return ((globalThis as any)[contextSymbol] ||= createContext<LayoutContext>({
    site: null!,
    page: null!,
  }));
}

let LayoutContext = getLayoutContext();

export interface LayoutProps {
  url: string;
  frontmatter: Record<string, any>;
  contents: string;
}

export let usePage = () => useContext(LayoutContext).page;
export let useSite = () => useContext(LayoutContext).site;

/**
 * @internal
 */
export function renderLayout(
  site: Site,
  page: Page,
  Layout: FunctionComponent<LayoutProps>,
) {
  return (
    "<!DOCTYPE html>" +
    renderToString(
      <LayoutContext.Provider value={{ site, page }}>
        <Layout
          url={site.url}
          contents={page.contents}
          frontmatter={page.frontmatter}
        />
      </LayoutContext.Provider>,
    )
  );
}

export let Scripts = () => {
  let page = usePage();

  return (
    <>
      {page.scripts.map(src => <script type="module" src={src} />)}
    </>
  );
}

export let LiveReload = () => {
  let site = useSite();

  if (site.env === "production") {
    return null;
  }

  return (
    <script
      dangerouslySetInnerHTML={{
        __html: `new WebSocket("ws://" + location.host).onmessage = () => location.reload()`,
      }}
    />
  );
};

export let Links = () => {
  let page = usePage();

  return (
    <>
      {page.styles.map(href => <link rel="stylesheet" href={href} />)}
    </>
  );
};

export let Contents: FunctionComponent<ComponentProps<"div">> = props => {
  let page = usePage();

  return (
    <div {...props} dangerouslySetInnerHTML={{ __html: page.contents }} />
  );
};

export let DefaultLayout: FunctionComponent<LayoutProps> = ({ url, frontmatter }) => {
  return (
    <html lang="en" dir="ltr">
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <link rel="canonical" href={url} />
        <title>{frontmatter.title}</title>
        <Links />
      </head>
      <body>
        <Nav />
        <h1>{frontmatter.title}</h1>
        {frontmatter.date && (
          <time class="date" dateTime={frontmatter.date}>
            {frontmatter.date}
          </time>
        )}

        <main>
          <Contents />
          {frontmatter.index && <Index />}
        </main>

        <Scripts />
        <LiveReload />
      </body>
    </html>
  );
};

function Index() {
  let site = useSite();
  return (
    <ul>
      {site.pages
        .filter(page => !page.frontmatter.index)
        .map(page => (
          <li>
            <a href={page.url}>{page.frontmatter.title}</a>
          </li>
        ))}
    </ul>
  );
}

function Nav() {
  let site = useSite();
  return (
    <nav>
      {site.pages
        .filter(page => !page.frontmatter.nav)
        .map(page => (
          <a href={page.url}>{page.frontmatter.title}</a>
        ))}
    </nav>
  );
}
