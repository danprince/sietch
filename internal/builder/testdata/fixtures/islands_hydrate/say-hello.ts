export function render({ name }) {
  return `Hello, ${name}!`;
}

export function hydrate(props, element) {
  element.innerHTML = render(props);
}
