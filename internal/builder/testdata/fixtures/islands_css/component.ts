import "./component.css";
export function render({ name }) {
  return `Hello, ${name}!`;
}

export function hydrate(props, element: HTMLElement) {
  element.innerHTML = render(props);
}
