export function onIdle(): Promise<void> {
  return new Promise(resolve => {
    if (typeof requestIdleCallback === "function") {
      requestIdleCallback(() => resolve());
    } else {
      setTimeout(() => resolve(), 200);
    }
  })
}

export function onVisible(element: HTMLElement): Promise<void> {
  return new Promise(resolve => {
    let observer = new IntersectionObserver(entries => {
      if (entries[0].isIntersecting) {
        resolve();
        observer.disconnect();
      }
    }, { threshold: [0] });

    observer.observe(element);
  })
}
