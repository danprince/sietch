import { h } from 'preact';
import { render } from 'preact-render-to-string';
import $ca from './Counter.tsx';
$elements['a'] = render(h($ca, {"count":1}));
import $cb from './Counter.tsx';
$elements['b'] = render(h($cb, {"count":3}));
import $cc from '../Timer.tsx';
$elements['c'] = render(h($cc, {}));
