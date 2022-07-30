---
title: Lisp in Your Language
permalink: /2015/09/09/lisp-in-your-language/index.html
---

I'm a fan of Lisp programming languages, but there's an incredible conceptual elegance that struggles to materialise as readable elegance for many unfamiliar programmers. The underlying concepts are incredibly simple, but the learning curve can represent a disproportionate challenge.

![Conceptual Elegance](https://i.imgur.com/UZzlqUy.jpg)

## Brief History

Lisp is a derivation of the phrase **Lis**t **P**rocessing. The fundamental idea of the language is that you represent your ideas and constructs as data structures, rather than with structured syntax. Specifically you represent them as lists.

```lisp
(print "Hello world!")
```

* Use `(` and `)` to denote lists
* Arguments are space separated
* First item is a function
* Remaining items are the arguments

Constructs you may be used to seeing implemented with special syntax or keywords suddenly become similar to the example above above.

```lisp
(if (= 5 5)
  (print "Sanity!")
  (print "Insanity!"))
```

`if` is just a special function that evaluates a condition, if that condition is found to be true, it evaluates the second argument otherwise it evaluates the third argument.

These functions are often known as _special forms_. Core bits of syntax are often implemented as special forms, but there's nothing particularly special about them. You can implement them yourself using macros. [Clojure][1] (like many Lisps) implements many of the core constructs [with macros][2].

We've been writing code to manipulate data for a long time now. When your code is also data, you can write code to manipulate code just as easily.

The essence of this wonder isn't Clojure though. It's not Racket or Scheme either. These are all just different incarnations of the code-as-data idea. These languages certainly aren't the only ones with functions and lists!

What if we could write code-as-data in our language of choice?

## An Experiment
There's a Lisp hidden in many popular programming languages, although it may take a bit of work to uncover it. You may have to do things you won't be proud of, but if you can think of a programming language with lists and [higher-order functions][3], then it will be there. Take Javascript, for example.

```lisp
(print "Hello world!")
```

What is stopping us from simply translating the syntax from the above example into Javascript?

```js
[alert, 'Hello world!']
```

Nothing, except it doesn't do much. It returns an array that contains a function and a string. Just the way Lisp wants. But our Javascript runtimes aren't expecting us to be writing code this way. If it was possible to ask them to try and execute all arrays as though they were functions, there would be chaos.

We're going to have to do a little bit of work to make this happen. Let's define an eval function which will interpret an expression.

```js
function eval(expression) {
  // the first item is the function
  var fn = expression[0];

  // the remaining items are the arguments
  var args = expression.slice(1);

  // call the function with these arguments
  return fn.apply(null, args);
}
```

And to see it in action:

```js
eval([alert, 'Hello world!']);
// alerts 'Hello world!'
```

That's it, we've implemented a (very minimal) Lisp. We can try out some other built-in functions too. From now on, the call to `eval` will be omitted from examples for brevity.

```js
[parseInt, '4.41'] // 4
[isNaN, 103]       // false
[btoa, 42]         // 'NDI='
```

There's a [good reason][4] why our eval function won't work if you try it with `console.log` or `document.write`, so stick to `alert` for now.

## Expressions All The Way Down

![Turtles All The Way Down](https://i.imgur.com/7cx4nBz.jpg)

From here on, we'll refer to the lists in our code as [expressions][8]. This helps distinguish them from list data structures. What happens when we try and evaluate an expression that already contains another expression?

```js
[alert, [prompt, "What is your name?"]]
```

We get an alert that tries to alert the inner expression as though it was an array. We need to make our eval function understand that if it finds an expression as an argument, it should evaluate it as code, not data.

```js
function eval(expression) {
  // the first item is the function
  var fn = expression[0];

  // the remaining items are the arguments
  var args = expression
    .slice(1)
    .map(function(arg) {
      // if this argument is an expression, treat it as code
      if(arg instanceof Array) {
        return eval(arg);
      } else {
        return arg;
      }
    });

  // call the function with these arguments
  return fn.apply(null, args);
}
```

Now we've got some recursion in the mix, we're getting somewhere. This function will evaluate every array it finds, no matter how deep into the structure.

```js
[alert, [prompt, "What is your name?"]]
```

## Syntax & Names
So far, so good, but how would we do Maths?

```js
[+, 5, 5]
```

Like it or not, this is definitely going to give you a syntax error.

One of the genuine benefits of picking a language that already understands Lisp is that the simplicity of the syntax leaves an abundance of characters to use as identifier names. For instance, in Clojure `+` is just the name of a function that happens to be responsible for adding numbers.

When we want to borrow these transcendental concepts in our syntax heavy languages, we have to do some extra work.

```js
function add(a, b) {
  return a + b;
}

[add, 5, 5] // 10
```

This is elegant for sure, but there's scope for more mischief here. Try this instead.

```js
['+', 5, 5] // Error: '+' is not a function
```

Let's define some native functions.

```js
var native = {
  '+': function(a, b) {
    return a + b;
  },
  '-': function(a, b) {
    return a - b;
  }
};

[native['+'], 5, 5] // 10
```

This ends up feeling verbose, but some tweaks can alleviate it. Pass your native object to eval as a second argument.

```js
function eval(expression, native) {
  // the first item is the function or it's name
  var fnName = expression[0];

  // resolve the function from native if necessary
  var fn = typeof fnName === 'string' ? native[fnName] : fnName;

  // the remaining items are the arguments
  var args = expression
    .slice(1)
    .map(function(arg) {
      // if this argument is an expression, treat it as code
      if(arg instanceof Array) {
        return eval(arg, native);
      } else {
        return arg;
      }
    });

  // call the function with these arguments
  return fn.apply(null, args);
}

['+', 5, 5] // 10
```

Hopefully, you're wondering why this doesn't feel like the zen of simplicity that is associated with Lisps. And you're right. It's not. But if you wanted simple, then you should ask yourself what on earth are you doing reading about implementing a makeshift lisp in an already confused programming language?

![Makeshift Lisp + Confused Programming Language](https://i.imgur.com/23mGq6v.jpg)

This is a sandbox for us to do unreasonable things in. Missing out on these kinds of hacks would be a wasted opportunity. Go ahead and implement `+`, `-`, `*`, `/`, `=` and any other operators you think might be useful as native functions. We'll use them later on.

## Variables
A language without variables would be difficult, so we'll implement them.

```js
function def(name, value) {
  window[name] = value;
}

[def, a, 5]
```

Our `def` function takes a variable name and a value to assign to it, then it binds it onto the `window` object—which is the global scope in Javascript. However, there's a real elephant in the expression. We aren't responsible for resolving the values of variables within the expression. The Javascript implementation is going to do that for us.

It will try to resolve the value of `a`. We haven't declared it, so it will throw an error. Or even worse, if we have declared it, but not initialised it, we'll end up with `undefined` as our name argument. Of course Javascript has an _excellent_ way of dealing with this. Coerce `undefined` to a string, then use it as a key all the same (oh, Javascript...).

Ah well. The obvious solution is to pass the name as a string instead.

```js
[def, 'a', 5]
[alert, ['+', a, a]]
```

Great, except it still doesn't work. The second expression is evaluated by the runtime before we get a chance to interpret the first. How did we solve this last time? Use strings instead.

## Scope

```js
[def, 'a', 5]
[alert, ['+', 'a', 'a']]
```

Now we have to try and resolve every string argument as a variable. We're also going to have do the same with functions, so that we can use variables as the first item in lists.

Let's bite the bullet and introduce a simple scope, then have all strings refer to values within it. If a string doesn't refer to a value, then we'll just use it's raw value.

Instead of accepting the `native` object as a second argument, accept a `scope` object instead. This way, we can pass our native object in as the root scope object and nothing will break.

```js
function eval(rawExpr, scope) {

  // if the expression isn't a list, just return it
  if(!(rawExpr instanceof Array)) {
    return rawExpr;
  }

  // use existing local scope or create a new one
  scope = scope || {};

  // resolve all our new string names from our scope
  var expression = rawExpr.map(function(symbol) {
    if(symbol in scope) {
      return scope[symbol];
    } else {
      return symbol;
    }
  });

  // the first item is the function
  var fn = expression[0];

  // the remaining items are the arguments
  var args = expression
    .slice(1)
    .map(function(arg) {
      // if this argument is an expression, treat it as code
      if(arg instanceof Array) {
        return eval(arg, scope);
      } else {
        return arg;
      }
    });

  // call the function with these arguments
  // and expose scope as this
  return fn.apply(scope, args);
}
```

We used the first argument of [`.apply`][5] to expose the scope as `this` to each of our functions. We'll define a new, native version of `def` to show `this` in action (excuse the pun).

```js
var native = {
  def: function(name, value) {
    return this[name] = value;
  },
  print: console.log.bind(console)
};
```

We can also add a `print` method, just in case you were fed up of using alert. Let's test that out.

```js
['print', ['def', 'a', 5]]
```

It may not be the most beautiful code you've ever seen, but it works.

## Special Forms

We've got evaluable expressions, but we don't have any way to control them. There's no sense of a conditional statement, a function, or even a way to execute multiple expressions at once.

Our eval function currently tries to interpret every expression it sees. We'll have to denote that some functions are special forms that will handle the evaluation of their own arguments.

```js
function SpecialForm(fn) {
  fn.__isSpecialForm__ = true;
  return fn;
}
```

Then we'll tweak the eval function, to prevent it from evaluating expressions that are arguments to a special form.

```js
// ...
// the first item is the function
var fn = expression[0];

// the remaining items are the arguments
var args = expression
  .slice(1)
  .map(function(arg) {
    // don't evaluate the expression if it is a special form!
    if(arg instanceof Array && (!fn.__isSpecialForm__) {
      return eval(arg, scope);
    } else {
      return arg;
    }
  });
// ...
```

## Do

Let's test out our new special forms and implement `do`. It evaluates all of its arguments, which allows us to evaluate multiple expressions in series.

In traditional Lisp:

```lisp
(do
  (print "Hello")
  (print "World!"))
```

We'll add it as a new native function.

```js
var native = {
  'do': SpecialForm(function() {
    var exprs = [].slice.call(arguments);
    return exprs.reduce(function(_, expr) {
      return eval(expr, this);
    }.bind(this), null);
  }
};
```

We can also do a nice trick with [reduce][9] to make sure that the value of the last expression is returned.

Lets translate the example above to our new syntax and watch it run.

```js
['do',
  ['print', 'Hello'],
  ['print', 'World!']]

// Hello
// World!
```

## If/Else
What good is a programming language without conditionals? The next challenge is implementing if statements. However—with our new special forms—it should be trivial.

```js
var native = {
  if: SpecialForm(function(condition, success, failure) {
    var passed = eval(condition, native, this);
    return eval(passed ? success : failure, native, this);
  }
};
```

That's it. `if/else` in 3 lines of code.

```js
['if', ['=', 3, 3],
  ['print', 'true'],
  ['print', 'false']]

// true
```

If this is your first time implementing a Lisp, this should be a special moment. You have implemented conditional control flow as data.

## Functions
Functions are the last hurdle between here and having a language that can actually do things. However, it's quite a hurdle.

Here's what they look like in more conventional Lisps.

```lisp
(def shout
  (fn [name planet]
    (print planet name)))
```

This is actually an _anonymous function_ being bound to a local variable with `def`. We already have an implementation of `def` so all we need now is an implementation for `fn`.

Let's break down the arguments to `fn`.

The first one is an list of arguments and the second one is the expression (or function body).

```js
var native = {
  fn: SpecialForm(function(defArgs, expr) {
    return function() {
      var callArgs = arguments;

      // create a distinct new scope
      var childScope = Object.create(this);

      // use the arguments definition to bind each call arg
      // to the appropriate scope variable.
      defArgs.forEach(function(argName, index) {
        childScope[argName] = callArgs[index];
      });

      // evalue the function body
      return eval(expr, code, childScope);
    }
  })
};
```

There it is. Dynamic binding into a lexical scope. Can we just take a moment to agree that prototypal inheritance rocks, too?

```js
['do',
  ['def', 'shout',
    ['fn', ['planet', 'greeting'],
      ['print', 'greeting', 'planet']]],
  ['shout', 'hello', 'world']]

// hello world
```

This could definitely be less verbose, so we can take a hint from some other Lisps and create `defn` too.

```js
var native = {
  defn: SpecialForm(function(name, args, expr) {
    var fn = native.fn.call(this, args, expr);
    return native.def.call(this, name, fn);
  })
};
```

We simply tie together our existing implementation of `def` with `fn`.

```js
['do',
  ['defn', 'shout', ['planet', 'greeting'],
    ['print', 'greeting', 'planet']],
  ['shout', 'hello', 'world']]

// hello world
```

Much better.

Once a language has functions, [the sky is the limit][6].

```js
["defn", "fib", ["n"],
  ["if", [">", "n", 1],
    ["+",
      ["fib", ["-", "n", 1]],
      ["fib", ["-", "n", 2]]],
    1]]
```

No self-respecting functional programming demo comes without a horribly inefficient demo of a non-memoized recursive Fibonnaci implementation. This one is no exception.

```js
["print", ["fib", 8]]
// 34
```

## Considerations
You might have noticed that our code is completely JSON compliant. We use primitives and lists. This means we can actually use JSON as source files for our language.

What? You mean we can embed a language with first class functions _inside JSON_? Yeah, we can.

Our language is still very short on the ground in terms of a standard library. We haven't really considered data structures, namespaces, exceptions, debugging or macros either.

## Conclusion
I'm putting together an implementation of this Javascript Lisp, along with a REPL and a growing set of native functions [on Github][7]. Feel free to use it as a reference. It's important to remember is that this is a toy—a sandbox for learning. It's not meant to be taken seriously and it certainly shouldn't be used in any real systems. It's inefficient and insecure.

Here's a short video of the REPL in action.
<script type="text/javascript" src="https://asciinema.org/a/09hjbv3sudn2iff6gh2gldawx.js" id="asciicast-09hjbv3sudn2iff6gh2gldawx" async></script>

More than anything else, implementing a programming language—no matter how small or strange—is a great way to learn about the language you implement it in. Language design in general is a fairly eye-opening experience and hopefully this has also helped open your eyes to the simple, yet powerful nature of Lisps.

I'll revisit this language again in the future, to talk through the process of implementing macros, then we'll move as much native code as possible inside the language.

Now open your editor and do this again in another language, then [tweet me][10] when you're done!

[1]: http://clojure.org/ "Clojure's Website"
[2]: http://clojure.org/macros "Clojure Macros"
[3]: https://en.wikipedia.org/wiki/Higher-order_function "Wikipedia Higher Order Functions"
[4]: http://stackoverflow.com/questions/12944987/abbreviating-console-log-in-javascript "Abbreviating console.log"
[5]: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Function/apply "MDN Function.prototype.apply"
[6]: http://www.inf.fu-berlin.de/lehre/WS03/alpi/lambda.pdf "Introduction to the Lambda Calculus"
[7]: https://github.com/danprince/ljsp "danprince/ljsp"
[8]: http://rosettacode.org/wiki/S-Expressions "S-Expressions"
[9]: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Array/Reduce "MDN Function.prototype.reduce"
[10]: https://twitter.com/workshydev "Tweet Me"
