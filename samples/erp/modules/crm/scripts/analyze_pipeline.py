def execute(bitcode, params):
    leads = bitcode.model("lead").search({
        "domain": [["status", "in", ["new", "contacted", "qualified", "proposal", "won"]]],
        "fields": ["id", "name", "status", "expected_revenue"],
    })

    if not isinstance(leads, list):
        leads = []

    total_revenue = sum(
        l.get("expected_revenue", 0) for l in leads if isinstance(l, dict)
    )
    won_count = sum(
        1 for l in leads if isinstance(l, dict) and l.get("status") == "won"
    )

    result = {
        "total_leads": len(leads),
        "total_revenue": total_revenue,
        "won_count": won_count,
        "conversion_rate": (won_count / len(leads) * 100) if leads else 0,
    }

    bitcode.log("info", "Pipeline analysis complete", result)
    return result
