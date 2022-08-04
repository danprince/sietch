export let render = ({ count }) => `<button>${count}</button>`;

export let hydrate = ({ count }, element: HTMLElement) => {
  let btn = element.querySelector("button")!;
  btn.onclick = () => btn.textContent = String(++count);
};
