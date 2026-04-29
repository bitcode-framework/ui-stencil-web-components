export default {
  async execute(bitcode, params) {
    const employees = await bitcode.model("employee").search({
      domain: [["active", "=", true]],
    });

    bitcode.log("info", "Weekly attendance report generated", {
      employeeCount: employees.length,
    });

    return { report: "generated", employeeCount: employees.length };
  },
};
