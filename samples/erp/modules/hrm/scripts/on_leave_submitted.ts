export default {
  async execute(bitcode, params) {
    const employeeId = params.employee_id || params.input?.employee_id;
    const days = params.days || params.input?.days || 0;

    if (employeeId) {
      const employee = await bitcode.model("employee").get(employeeId);
      if (employee?.manager_id) {
        await bitcode.notify.send({
          to: employee.manager_id,
          title: "Leave Request",
          message: `${employee.name} has submitted a leave request for ${days} days.`,
          type: "info",
        });
      }
    }

    bitcode.log("info", "Leave request submitted, manager notified", { employeeId, days });
    return { notified: true };
  },
};
