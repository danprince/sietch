// @ts-nocheck
/** @jsx h */
/** @jsxFrag Fragment */

import "./_counter.css";
import { h } from "preact";
import { useState, useEffect } from "preact/hooks"

let Counter = ({ count: initialCount = 0 }) => {
  let [count, setCount] = useState(initialCount);

  return (
    <div>
      <button onClick={() => setCount(count + 1)}>{count}</button>
    </div>
  );
}

export default Counter;

