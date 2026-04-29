def execute(bitcode, params):
    employee_id = params.get("input", {}).get("employee_id", "")
    total_annual = 12

    approved_leaves = bitcode.model("leave_request").search({
        "domain": [
            ["employee_id", "=", employee_id],
            ["status", "=", "approved"],
        ],
        "fields": ["days"],
    })

    used = sum(l.get("days", 0) for l in (approved_leaves or []) if isinstance(l, dict))
    remaining = total_annual - used

    return {
        "employee_id": employee_id,
        "total_annual": total_annual,
        "used": used,
        "remaining": max(0, remaining),
        "can_take_leave": remaining > 0,
    }
