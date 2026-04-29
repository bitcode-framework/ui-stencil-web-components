export default {
  async execute(bitcode, params) {
    const lead = params.input;
    bitcode.log("info", `Deal Lost: ${lead.name} - Reason: ${lead.lost_reason}`);
    return { success: true };
  },
};
