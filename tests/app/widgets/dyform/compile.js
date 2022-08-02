/**
 * The DSL compiler.
 * Translate the customize DSL to Models, Processes, Flows, Tables, etc.
 */

/**
 * Compile
 * Translate or extend the customize DSL
 * @param {*} dsl
 */
function Compile(name, source) {
  let dsl = {};
  return dsl;
}

/**
 * Prepare
 * When the yao server started, the function will be called.
 * For preparing the sources the widget need.
 * @param {DSL} dsl
 */
function Prepare(dsl) {}

/**
 * Load
 * When the widget instance are loaded, the function will be called.
 * For preparing the sources the widget need.
 * @param {DSL} dsl
 */
function Load(dsl) {}

/**
 * Migrate
 * When the migrate command executes, the function will be called
 * @param {DSL} dsl
 * @param {bool} force
 */
function Migrate(dsl, force) {}
