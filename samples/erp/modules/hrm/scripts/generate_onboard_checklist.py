def execute(bitcode, params):
    employee = params.get("input", {})
    name = employee.get("name", "Unknown")
    department_id = employee.get("department_id", "")

    dept_name = "general"
    if department_id:
        dept = bitcode.model("department").get(department_id)
        if dept and isinstance(dept, dict):
            dept_name = dept.get("name", "general")

    checklist = [
        {"task": "Create email account", "done": False},
        {"task": "Issue ID badge", "done": False},
        {"task": "Setup workstation", "done": False},
        {"task": "Assign mentor", "done": False},
        {"task": "Department orientation: {}".format(dept_name), "done": False},
        {"task": "HR policy briefing", "done": False},
        {"task": "IT security training", "done": False},
    ]

    bitcode.log("info", "Onboarding checklist generated for {}".format(name))
    return {"employee": name, "checklist": checklist, "total_tasks": len(checklist)}
