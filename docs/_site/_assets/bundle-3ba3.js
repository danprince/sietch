// snippets/counter.ts
var hydrate = ({ count }, element) => {
  let btn = element.querySelector("button");
  btn.onclick = () => btn.textContent = String(++count);
};

// browser-pages:3ba3?browser
hydrate({ "count": 0 }, document.getElementById("3ba3_1"));
//# sourceMappingURL=bundle-3ba3.js.map
