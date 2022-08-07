type BaseProps = Record<string, any>;

export function jsx(el: string, props: BaseProps | null): string
export function jsx<P extends BaseProps>(el: (props: P) => string, props: P): string
export function jsx(el: string | Function, props: BaseProps | null): string {
  if (typeof el === "function") return el(props);
  let { children, ...attrs } = props || {};

  let attributes = Object
    .entries(attrs)
    .map(([k, v]) => `${k}="${v}"`)
    .join(" ");

  let innerHtml = Array.isArray(children) ? children.join("\n") : String(children);
  return `<${el} ${attributes}>${innerHtml}</${el}>`
}

export function Fragment(props: BaseProps) {
  return props.children;
}

export {
  jsx as jsxs,
  jsx as jsxDEV,
}
