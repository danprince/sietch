import { OnLoadResult, Plugin } from "esbuild";

/**
 * @internal
 */
export function virtualModulesPlugin({
  filter,
  modules,
}: {
  filter: RegExp;
  modules: Record<string, OnLoadResult>;
}): Plugin {
  return {
    name: "Virtual Modules Plugin",
    setup(build) {
      build.onResolve({ filter }, args => {
        return {
          namespace: "virtual",
          path: args.path,
        };
      });

      build.onLoad({ namespace: "virtual", filter }, args => {
        return modules[args.path];
      });
    },
  };
}
