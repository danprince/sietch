// @ts-nocheck
/** @jsx React.createElement */
/** @jsxFrag React.Fragment */
import "./_counter.css";
import React, { useState } from "react";

let Counter = ({ count: initialCount = 0 }) => {
  let [count, setCount] = useState(initialCount + 5);
  return (
    <div>
      <button onClick={() => setCount(count + 1)}>{count}</button>
    </div>
  );
}

export default Counter;
