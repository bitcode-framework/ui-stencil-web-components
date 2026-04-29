export default {
  async execute(bitcode, params) {
    const leadId = params.lead_id || params.input?.id;
    await bitcode.notify.send({
      to: "manager",
      title: "Lead Qualified",
      message: `Lead ${leadId} has been qualified. Manager review needed.`,
      type: "info",
    });
    bitcode.log("info", "Manager notified for lead qualification", { leadId });
    return { notified: true };
  },
};
