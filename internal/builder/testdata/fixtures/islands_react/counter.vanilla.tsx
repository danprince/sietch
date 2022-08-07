/** @jsxImportSource react */
import { useState } from "react";
import { hydrate as _hydrate } from "react-dom/client";

// https://github.com/anonyco/FastestSmallestTextEncoderDecoder/issues/18
import "fastestsmallesttextencoderdecoder/EncoderDecoderTogether.min";
import { renderToString } from "react-dom/server.browser";

let Counter = ({ count: init = 0 }) => {
  let [count, setCount] = useState(init);
  return <button onClick={() => setCount(count + 1)}>{count}</button>;
}

export function render(props) {
  return renderToString(<Counter {...props} />);
}

export function hydrate(props, element) {
  hydrate(<Counter {...props} />, element);
}
