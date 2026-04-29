export default {
  async execute(bitcode, params) {
    const leave = params.input;
    const days = leave?.days || 0;
    const employeeId = leave?.employee_id;

    if (employeeId && days > 0) {
      const employee = await bitcode.model("employee").get(employeeId);
      if (employee) {
        const newBalance = (employee.leave_balance || 0) - days;
        await bitcode.model("employee").write(employeeId, {
          leave_balance: newBalance,
        });
      }
    }

    bitcode.log("info", "Leave approved, balance updated", { employeeId, days });
    return { balance_updated: true };
  },
};
