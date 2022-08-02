/**
 * Export the widget processes
 * Export() defined witch function in this file will be registered as YAO PROCESS
 * The process name is <WIDGET NAME>.<INSTANCE NAME>.<FUNCTION NAME>
 * The processes can be used in compile.js and export.js DIRECTLY
 */

/**
 * SchemaSave
 * Save the schema of dyform
 * @name: dyform.<INSTANCE>.SchemaSave
 * @param {*} payload
 */
function SchemaSave(payload) {
  return payload;
}

/**
 * Export processes
 */
function Export() {
  return { Save: "SaveSchema" };
}
