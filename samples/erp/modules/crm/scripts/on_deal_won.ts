export default {
  async execute(bitcode, params) {
    const lead = params.input;

    await bitcode.model("activity").create({
      lead_id: lead.id,
      type: "task",
      summary: "Send welcome package to new client",
    });

    await bitcode.email.send({
      to: "manager@company.com",
      subject: "Deal Won: " + lead.name,
      body: "<h1>Revenue: $" + lead.expected_revenue + "</h1>",
    });

    bitcode.log("info", "Deal won processed", { leadId: lead.id });
    return { success: true, message: "Deal won notification sent" };
  },
};
