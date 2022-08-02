/**
 * Export Models, APIs, Flows, Tables, Tasks, Schedules, etc.
 */

/**
 * Export APIs
 * @param {*} dsl
 * @returns
 */
function APIs(dsl) {
  return [{}];
}

/**
 * Export Models
 * @param {*} dsl
 * @returns
 */
function Models(dsl) {
  return { dyform: dyformModel() };
}

/**
 * Export Flows
 * @param {*} dsl
 * @returns
 */
function Flows(dsl) {
  return [{}];
}

/**
 * Export Tables
 * @param {*} dsl
 * @returns
 */
function Tables(dsl) {
  return [{}];
}

/**
 * Export Tasks
 * @param {*} dsl
 * @returns
 */
function Tasks(dsl) {
  return [{}];
}

/**
 * Export Schedules
 * @param {*} dsl
 * @returns
 */
function Schedules(dsl) {
  return [{}];
}

function dyformModel() {
  return {
    table: { name: "dyform" },
    columns: [
      { label: "DYFORM ID", name: "id", type: "ID" },
      { label: "SN", name: "sn", type: "string", length: 20, unique: true },
      { label: "NAME", name: "name", type: "string", length: 200, index: true },
      { label: "SOURCE", name: "source", type: "JSON", nullable: true },
      {
        label: "TITLE",
        name: "title",
        type: "string",
        length: 200,
        index: true,
      },
    ],
    indexes: [],
  };
}
