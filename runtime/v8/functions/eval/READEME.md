# Eval Function

The `Eval` function allows JavaScript code execution from strings within the V8 runtime environment. This is useful for dynamically generating and executing code at runtime.

## Usage

```javascript
// Syntax
Eval(codeString, ...args);
```

### Parameters

- `codeString` (string): A JavaScript code string to evaluate. It should contain a function declaration.
- `...args`: Optional parameters to pass to the evaluated function.

### Return Value

Returns the result of executing the evaluated function.

## Examples

### Basic Example

```javascript
// Evaluate a simple addition function
const sum = Eval("function add(a, b) { return a + b; }", 5, 3);
console.log(sum); // Output: 8
```

### Accessing Context Data

When running in the Yao runtime environment, the evaluated code can access the context data:

```javascript
// Assume __yao_data.DATA.user = "World"
const greeting = Eval(
  "function greet(prefix) { return prefix + ', ' + __yao_data.DATA.user + '!'; }",
  "Hello"
);
console.log(greeting); // Output: "Hello, World!"
```

## Implementation Details

The Eval function:

1. Takes a JavaScript code string, typically containing a function definition
2. Converts the function definition to an arrow function format for execution
3. Compiles and runs the script in the V8 context
4. Passes any additional arguments to the evaluated function
5. Returns the result of the function execution

## Notes

- The evaluated code has access to the current V8 context, including any global variables
- For security reasons, be careful when evaluating user-provided code
- Performance may be impacted when frequently evaluating code as compilation occurs each time
