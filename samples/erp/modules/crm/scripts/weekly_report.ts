export default {
  async execute(bitcode, params) {
    const leads = await bitcode.model("lead").search({
      domain: [["status", "in", ["qualified", "proposal"]]],
    });

    const totalRevenue = leads.reduce(
      (sum, l) => sum + (l.expected_revenue || 0),
      0
    );

    bitcode.log("info", "Weekly pipeline report generated", {
      leadCount: leads.length,
      totalRevenue,
    });

    return { report: "generated", leadCount: leads.length, totalRevenue };
  },
};
