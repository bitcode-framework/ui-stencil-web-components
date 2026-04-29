export default {
  async execute(bitcode, params) {
    const staleLeads = await bitcode.model("lead").search({
      domain: [
        ["status", "not in", ["won", "lost"]],
      ],
    });

    for (const lead of staleLeads) {
      if (lead.assigned_to) {
        await bitcode.notify.send({
          to: lead.assigned_to,
          title: "Stale Lead Reminder",
          message: `Lead "${lead.name}" has had no activity recently.`,
          type: "warning",
        });
      }
    }

    bitcode.log("info", "Stale leads check completed", {
      staleCount: staleLeads.length,
    });
    return { checked: true, staleCount: staleLeads.length };
  },
};
