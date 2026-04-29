export default {
  async execute(bitcode, params) {
    const employeeId = params.employee_id || params.input?.id;

    if (employeeId) {
      const employee = await bitcode.model("employee").get(employeeId);
      if (employee?.email) {
        await bitcode.email.send({
          to: employee.email,
          subject: "Congratulations on your promotion!",
          body: `<h1>Congratulations, ${employee.name}!</h1><p>You have been promoted.</p>`,
        });
      }
    }

    bitcode.log("info", "Promotion processed", { employeeId });
    return { notified: true };
  },
};
