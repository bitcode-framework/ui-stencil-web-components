/**
 * BitCode Components Demo — Mock API
 *
 * Intercepts fetch() to serve realistic CRM/ERP data for ALL data-dependent
 * components (datatable, views, relational fields, search, timeline, etc.).
 *
 * Loaded AFTER the Stencil bundle in index.html.
 */
(function () {
  'use strict';

  // ─── UUID helper ────────────────────────────────────────────────
  let _seq = 1;
  function uuid() {
    return 'demo-' + String(_seq++).padStart(6, '0');
  }

  // ─── CONTACTS (used by view-list, view-form, view-activity, search, timeline, chatter, activity) ──
  var contacts = [
    { id: uuid(), name: 'Andi Pratama', email: 'andi@example.com', phone: '+628123000001', status: 'active', company: 'PT Maju Jaya', city: 'Jakarta', created_at: '2025-12-01T09:00:00Z', description: 'Key account manager' },
    { id: uuid(), name: 'Siti Nurhaliza', email: 'siti@example.com', phone: '+628123000002', status: 'active', company: 'CV Berkah', city: 'Bandung', created_at: '2025-12-02T10:30:00Z', description: 'New prospect from expo' },
    { id: uuid(), name: 'Budi Santoso', email: 'budi@example.com', phone: '+628123000003', status: 'inactive', company: 'PT Teknologi Indonesia', city: 'Surabaya', created_at: '2025-12-03T14:15:00Z', description: 'Former client, re-engage Q2' },
    { id: uuid(), name: 'Dewi Lestari', email: 'dewi@example.com', phone: '+628123000004', status: 'active', company: 'PT Global Medika', city: 'Medan', created_at: '2025-12-04T08:45:00Z', description: 'Healthcare sector lead' },
    { id: uuid(), name: 'Reza Firmansyah', email: 'reza@example.com', phone: '+628123000005', status: 'active', company: 'Startup Hub', city: 'Yogyakarta', created_at: '2025-12-05T11:00:00Z', description: 'Startup ecosystem partner' },
    { id: uuid(), name: 'Maya Putri', email: 'maya@example.com', phone: '+628123000006', status: 'active', company: 'PT Kreatif Digital', city: 'Jakarta', created_at: '2025-12-06T16:20:00Z', description: 'Digital agency owner' },
    { id: uuid(), name: 'Hendra Wijaya', email: 'hendra@example.com', phone: '+628123000007', status: 'lead', company: 'PT Sinar Abadi', city: 'Semarang', created_at: '2025-12-07T09:30:00Z', description: 'Interested in ERP package' },
    { id: uuid(), name: 'Lisa Permata', email: 'lisa@example.com', phone: '+628123000008', status: 'active', company: 'PT Retail Nusantara', city: 'Bali', created_at: '2025-12-08T13:00:00Z', description: 'Retail chain operations' },
    { id: uuid(), name: 'Agus Setiawan', email: 'agus@example.com', phone: '+628123000009', status: 'inactive', company: 'CV Mandiri', city: 'Makassar', created_at: '2025-12-09T07:45:00Z', description: 'Construction contractor' },
    { id: uuid(), name: 'Rina Wulandari', email: 'rina@example.com', phone: '+628123000010', status: 'active', company: 'PT Edukasi Bangsa', city: 'Jakarta', created_at: '2025-12-10T15:30:00Z', description: 'Education technology' },
    { id: uuid(), name: 'Tommy Kurniawan', email: 'tommy@example.com', phone: '+628123000011', status: 'lead', company: 'PT Foodindo', city: 'Bandung', created_at: '2025-12-11T10:00:00Z', description: 'F&B industry' },
    { id: uuid(), name: 'Sari Indah', email: 'sari@example.com', phone: '+628123000012', status: 'active', company: 'PT Logistik Express', city: 'Surabaya', created_at: '2025-12-12T12:15:00Z', description: 'Logistics & warehousing' },
  ];

  // ─── LEADS (used by view-kanban) ────────────────────────────────
  var leads = [
    { id: uuid(), name: 'Website Redesign', partner: 'PT Maju Jaya', stage: 'new', priority: 'high', expected_revenue: 25000000, probability: 20, user: 'Andi Pratama', created_at: '2025-12-01T09:00:00Z' },
    { id: uuid(), name: 'ERP Implementation', partner: 'CV Berkah', stage: 'new', priority: 'medium', expected_revenue: 150000000, probability: 15, user: 'Siti Nurhaliza', created_at: '2025-12-02T10:00:00Z' },
    { id: uuid(), name: 'Mobile App Development', partner: 'Startup Hub', stage: 'qualified', priority: 'high', expected_revenue: 75000000, probability: 40, user: 'Reza Firmansyah', created_at: '2025-12-03T11:00:00Z' },
    { id: uuid(), name: 'Cloud Migration', partner: 'PT Teknologi Indonesia', stage: 'qualified', priority: 'medium', expected_revenue: 100000000, probability: 35, user: 'Budi Santoso', created_at: '2025-12-04T08:00:00Z' },
    { id: uuid(), name: 'Inventory System', partner: 'PT Retail Nusantara', stage: 'proposition', priority: 'high', expected_revenue: 50000000, probability: 60, user: 'Lisa Permata', created_at: '2025-12-05T14:00:00Z' },
    { id: uuid(), name: 'CRM Setup', partner: 'PT Global Medika', stage: 'proposition', priority: 'low', expected_revenue: 30000000, probability: 55, user: 'Dewi Lestari', created_at: '2025-12-06T09:00:00Z' },
    { id: uuid(), name: 'Data Analytics Platform', partner: 'PT Edukasi Bangsa', stage: 'won', priority: 'high', expected_revenue: 200000000, probability: 100, user: 'Rina Wulandari', created_at: '2025-12-07T10:00:00Z' },
    { id: uuid(), name: 'HR System', partner: 'PT Kreatif Digital', stage: 'won', priority: 'medium', expected_revenue: 45000000, probability: 100, user: 'Maya Putri', created_at: '2025-12-08T11:00:00Z' },
    { id: uuid(), name: 'E-Commerce Integration', partner: 'PT Foodindo', stage: 'lost', priority: 'medium', expected_revenue: 80000000, probability: 0, user: 'Tommy Kurniawan', created_at: '2025-12-09T13:00:00Z' },
    { id: uuid(), name: 'Security Audit', partner: 'PT Sinar Abadi', stage: 'lost', priority: 'low', expected_revenue: 20000000, probability: 0, user: 'Hendra Wijaya', created_at: '2025-12-10T15:00:00Z' },
  ];

  // ─── KANBAN BOARD (enterprise kanban demo) ─────────────────────
  var kbUsers = [
    { id: 'u1', name: 'Andi Pratama', avatar: '' },
    { id: 'u2', name: 'Siti Nurhaliza', avatar: '' },
    { id: 'u3', name: 'Budi Santoso', avatar: '' },
    { id: 'u4', name: 'Dewi Lestari', avatar: '' },
    { id: 'u5', name: 'Reza Firmansyah', avatar: '' },
  ];
  var kbCards = [
    { id: uuid(), name: 'Design landing page', description: 'Create a modern, responsive landing page with hero section and feature highlights.', stage: 'todo', priority: 'high', start_date: '2026-05-25', assignees: [kbUsers[0], kbUsers[1]], labels: [{ id: 'l1', name: 'Design', color: '#8b5cf6' }, { id: 'l2', name: 'Frontend', color: '#3b82f6' }], due_date: '2026-06-15', position: 0, comments_count: 3, attachments_count: 2, subtasks: [{ id: 's1', title: 'Wireframe mockup', done: true }, { id: 's2', title: 'High-fidelity design', done: false }, { id: 's3', title: 'Responsive breakpoints', done: false }] },
    { id: uuid(), name: 'Setup CI/CD pipeline', description: 'Configure GitHub Actions for automated testing and deployment to staging.', stage: 'todo', priority: 'medium', start_date: '2026-06-01', assignees: [kbUsers[2]], labels: [{ id: 'l3', name: 'DevOps', color: '#10b981' }], due_date: '2026-06-20', position: 1, comments_count: 1, attachments_count: 0 },
    { id: uuid(), name: 'Implement auth module', description: 'JWT-based auth with refresh tokens, 2FA support, and session management.', stage: 'in_progress', priority: 'critical', start_date: '2026-05-20', assignees: [kbUsers[0], kbUsers[4]], labels: [{ id: 'l4', name: 'Backend', color: '#f59e0b' }, { id: 'l5', name: 'Security', color: '#ef4444' }], due_date: '2026-06-10', position: 0, comments_count: 7, attachments_count: 1, subtasks: [{ id: 's4', title: 'Login endpoint', done: true }, { id: 's5', title: 'Register endpoint', done: true }, { id: 's6', title: '2FA email OTP', done: false }, { id: 's7', title: 'Session management', done: false }] },
    { id: uuid(), name: 'Database schema design', description: 'Design normalized schema for CRM module with proper indexing.', stage: 'in_progress', priority: 'high', start_date: '2026-05-18', assignees: [kbUsers[2]], labels: [{ id: 'l4', name: 'Backend', color: '#f59e0b' }], position: 1, comments_count: 2, attachments_count: 3 },
    { id: uuid(), name: 'Unit tests for API', description: 'Achieve 80% coverage on REST API endpoints.', stage: 'review', priority: 'medium', start_date: '2026-05-28', assignees: [kbUsers[1], kbUsers[3]], labels: [{ id: 'l6', name: 'Testing', color: '#06b6d4' }], due_date: '2026-06-12', position: 0, comments_count: 4, attachments_count: 0, subtasks: [{ id: 's8', title: 'Auth tests', done: true }, { id: 's9', title: 'CRUD tests', done: true }, { id: 's10', title: 'Edge case tests', done: true }] },
    { id: uuid(), name: 'API documentation', description: 'OpenAPI 3.0 spec with examples and error codes.', stage: 'review', priority: 'low', start_date: '2026-06-02', assignees: [kbUsers[4]], labels: [{ id: 'l7', name: 'Docs', color: '#6b7280' }], position: 1, comments_count: 1, attachments_count: 1 },
    { id: uuid(), name: 'User registration flow', description: 'Complete registration with email verification and welcome email.', stage: 'done', priority: 'medium', start_date: '2026-05-10', assignees: [kbUsers[0], kbUsers[1]], labels: [{ id: 'l4', name: 'Backend', color: '#f59e0b' }, { id: 'l2', name: 'Frontend', color: '#3b82f6' }], position: 0, comments_count: 5, attachments_count: 2, due_date_complete: true, subtasks: [{ id: 's11', title: 'Form validation', done: true }, { id: 's12', title: 'Email verification', done: true }, { id: 's13', title: 'Welcome email', done: true }] },
    { id: uuid(), name: 'Project scaffolding', description: 'Initialize monorepo with shared configs.', stage: 'done', priority: 'low', start_date: '2026-05-01', assignees: [kbUsers[2]], labels: [{ id: 'l3', name: 'DevOps', color: '#10b981' }], position: 1, comments_count: 2, attachments_count: 0, due_date_complete: true },
  ];
  var kbComments = [
    { id: 'c1', card_id: kbCards[0].id, body: 'I started working on the wireframes. @budi can you review once done?', user: kbUsers[0], created_at: '2026-05-20T09:00:00Z', attachments: [] },
    { id: 'c2', card_id: kbCards[0].id, body: 'Looks good! Let\'s also consider dark mode from the start.', user: kbUsers[2], created_at: '2026-05-20T14:30:00Z', attachments: [] },
    { id: 'c3', card_id: kbCards[0].id, body: 'Agreed. I\'ll add dark mode toggle to the design spec. @siti can you help with color tokens?', user: kbUsers[0], created_at: '2026-05-21T08:15:00Z', attachments: [] },
    { id: 'c4', card_id: kbCards[2].id, body: 'JWT implementation is done. Moving to 2FA next. @reza FYI', user: kbUsers[0], created_at: '2026-05-22T10:00:00Z', attachments: [] },
    { id: 'c5', card_id: kbCards[2].id, body: 'Found an edge case with expired refresh tokens. Working on a fix.', user: kbUsers[4], created_at: '2026-05-22T16:45:00Z', attachments: [] },
    { id: 'c6', card_id: kbCards[2].id, body: 'Fixed. Also added rate limiting on the refresh endpoint.', user: kbUsers[4], created_at: '2026-05-23T09:20:00Z', attachments: [] },
    { id: 'c7', card_id: kbCards[2].id, body: 'Security review passed! Ready for QA.', user: kbUsers[0], created_at: '2026-05-23T15:00:00Z', attachments: [] },
  ];
  var kbActivities = [
    { id: 'a1', card_id: kbCards[0].id, action: 'created this card', user: kbUsers[0], created_at: '2026-05-19T08:00:00Z' },
    { id: 'a2', card_id: kbCards[0].id, action: 'moved from', detail: 'In Progress → To Do', user: kbUsers[0], created_at: '2026-05-20T09:00:00Z' },
    { id: 'a3', card_id: kbCards[2].id, action: 'changed priority to', detail: 'Critical', user: kbUsers[0], created_at: '2026-05-22T10:00:00Z' },
    { id: 'a4', card_id: kbCards[6].id, action: 'completed all subtasks', user: kbUsers[1], created_at: '2026-05-23T11:00:00Z' },
  ];

  // ─── EVENTS (used by view-calendar) ─────────────────────────────
  var events = [
    { id: uuid(), name: 'Team Standup', start: '2025-12-15T09:00:00', end: '2025-12-15T09:30:00', color: '#4f46e5' },
    { id: uuid(), name: 'Client Meeting — PT Maju Jaya', start: '2025-12-16T10:00:00', end: '2025-12-16T11:30:00', color: '#10b981' },
    { id: uuid(), name: 'Sprint Review', start: '2025-12-17T14:00:00', end: '2025-12-17T15:00:00', color: '#f59e0b' },
    { id: uuid(), name: 'Product Demo', start: '2025-12-18T11:00:00', end: '2025-12-18T12:00:00', color: '#3b82f6' },
    { id: uuid(), name: 'Year-End Review', start: '2025-12-20T09:00:00', end: '2025-12-20T17:00:00', color: '#ef4444' },
    { id: uuid(), name: 'Training Workshop', start: '2025-12-22T13:00:00', end: '2025-12-22T16:00:00', color: '#8b5cf6' },
    { id: uuid(), name: 'Strategy Meeting', start: '2025-12-23T10:00:00', end: '2025-12-23T11:00:00', color: '#4f46e5' },
    { id: uuid(), name: 'Quarterly Planning', start: '2026-01-05T09:00:00', end: '2026-01-05T17:00:00', color: '#10b981' },
  ];

  // ─── TASKS (used by view-gantt) ─────────────────────────────────
  var t1 = uuid(), t2 = uuid(), t3 = uuid(), t4 = uuid(), t5 = uuid(), t6 = uuid(), t7 = uuid();
  var tasks = [
    { id: t1, name: 'Phase 1 — Planning', start_date: '2025-12-01', end_date: '2025-12-10', progress: 1, parent: '0', type: 'project' },
    { id: uuid(), name: 'Requirements Gathering', start_date: '2025-12-01', end_date: '2025-12-05', progress: 1, parent: t1 },
    { id: uuid(), name: 'System Design', start_date: '2025-12-04', end_date: '2025-12-10', progress: 1, parent: t1 },
    { id: t2, name: 'Phase 2 — Development', start_date: '2025-12-08', end_date: '2025-12-22', progress: 0.45, parent: '0', type: 'project' },
    { id: t4, name: 'Backend Development', start_date: '2025-12-08', end_date: '2025-12-20', progress: 0.65, parent: t2 },
    { id: t5, name: 'Frontend Development', start_date: '2025-12-10', end_date: '2025-12-22', progress: 0.40, parent: t2 },
    { id: uuid(), name: 'API Integration', start_date: '2025-12-15', end_date: '2025-12-22', progress: 0.20, parent: t2 },
    { id: t3, name: 'Phase 3 — Delivery', start_date: '2025-12-20', end_date: '2026-01-05', progress: 0, parent: '0', type: 'project' },
    { id: uuid(), name: 'Testing & QA', start_date: '2025-12-20', end_date: '2025-12-28', progress: 0, parent: t3 },
    { id: t6, name: 'Deployment', start_date: '2025-12-28', end_date: '2025-12-31', progress: 0, parent: t3, type: 'milestone' },
    { id: t7, name: 'User Training', start_date: '2026-01-02', end_date: '2026-01-05', progress: 0, parent: t3 },
  ];

  // ─── BRANCHES (used by view-map) ────────────────────────────────
  var branches = [
    { id: uuid(), name: 'Jakarta HQ', location: { lat: -6.2088, lng: 106.8456 }, address: 'Jl. Sudirman Kav. 52-53', manager: 'Andi Pratama' },
    { id: uuid(), name: 'Bandung Office', location: { lat: -6.9175, lng: 107.6191 }, address: 'Jl. Dago No. 88', manager: 'Siti Nurhaliza' },
    { id: uuid(), name: 'Surabaya Branch', location: { lat: -7.2575, lng: 112.7521 }, address: 'Jl. Tunjungan No. 65', manager: 'Budi Santoso' },
    { id: uuid(), name: 'Medan Branch', location: { lat: 3.5952, lng: 98.6722 }, address: 'Jl. Gatot Subroto No. 12', manager: 'Dewi Lestari' },
    { id: uuid(), name: 'Yogyakarta Office', location: { lat: -7.7956, lng: 110.3695 }, address: 'Jl. Malioboro No. 33', manager: 'Reza Firmansyah' },
    { id: uuid(), name: 'Bali Office', location: { lat: -8.6705, lng: 115.2126 }, address: 'Jl. Sunset Road No. 100', manager: 'Lisa Permata' },
  ];

  // ─── CATEGORIES (used by view-tree) ─────────────────────────────
  var catRoot1 = uuid();
  var catRoot2 = uuid();
  var catRoot3 = uuid();
  var categories = [
    { id: catRoot1, name: 'Electronics', parent_id: null },
    { id: uuid(), name: 'Laptops', parent_id: catRoot1 },
    { id: uuid(), name: 'Desktops', parent_id: catRoot1 },
    { id: uuid(), name: 'Accessories', parent_id: catRoot1 },
    { id: catRoot2, name: 'Furniture', parent_id: null },
    { id: uuid(), name: 'Office Chairs', parent_id: catRoot2 },
    { id: uuid(), name: 'Desks', parent_id: catRoot2 },
    { id: uuid(), name: 'Cabinets', parent_id: catRoot2 },
    { id: catRoot3, name: 'Services', parent_id: null },
    { id: uuid(), name: 'Consulting', parent_id: catRoot3 },
    { id: uuid(), name: 'Implementation', parent_id: catRoot3 },
    { id: uuid(), name: 'Training', parent_id: catRoot3 },
  ];

  // ─── SALES (used by view-report) ────────────────────────────────
  var sales = [
    { id: uuid(), name: 'INV-2025-001', product: 'ERP License', customer: 'PT Maju Jaya', region: 'West', quantity: 1, unit_price: 150000000, amount: 150000000, date: '2025-12-01' },
    { id: uuid(), name: 'INV-2025-002', product: 'CRM Setup', customer: 'CV Berkah', region: 'West', quantity: 1, unit_price: 45000000, amount: 45000000, date: '2025-12-02' },
    { id: uuid(), name: 'INV-2025-003', product: 'Consulting', customer: 'PT Global Medika', region: 'North', quantity: 10, unit_price: 2500000, amount: 25000000, date: '2025-12-03' },
    { id: uuid(), name: 'INV-2025-004', product: 'Mobile App', customer: 'Startup Hub', region: 'Central', quantity: 1, unit_price: 75000000, amount: 75000000, date: '2025-12-04' },
    { id: uuid(), name: 'INV-2025-005', product: 'Training', customer: 'PT Edukasi Bangsa', region: 'West', quantity: 5, unit_price: 5000000, amount: 25000000, date: '2025-12-05' },
    { id: uuid(), name: 'INV-2025-006', product: 'Cloud Hosting', customer: 'PT Kreatif Digital', region: 'West', quantity: 12, unit_price: 2000000, amount: 24000000, date: '2025-12-06' },
    { id: uuid(), name: 'INV-2025-007', product: 'ERP License', customer: 'PT Logistik Express', region: 'East', quantity: 1, unit_price: 120000000, amount: 120000000, date: '2025-12-07' },
    { id: uuid(), name: 'INV-2025-008', product: 'Consulting', customer: 'PT Sinar Abadi', region: 'Central', quantity: 8, unit_price: 2500000, amount: 20000000, date: '2025-12-08' },
    { id: uuid(), name: 'INV-2025-009', product: 'Security Audit', customer: 'PT Retail Nusantara', region: 'East', quantity: 1, unit_price: 35000000, amount: 35000000, date: '2025-12-09' },
    { id: uuid(), name: 'INV-2025-010', product: 'Data Analytics', customer: 'PT Edukasi Bangsa', region: 'West', quantity: 1, unit_price: 200000000, amount: 200000000, date: '2025-12-10' },
  ];

  // ─── CONTACT AUDIT (used by bc-timeline — appends _audit to model) ──
  var contactAudits = [
    { id: uuid(), field: 'status', old_value: 'lead', new_value: 'active', user: 'Andi Pratama', created_at: '2025-12-05T10:00:00Z' },
    { id: uuid(), field: 'company', old_value: 'Freelance', new_value: 'PT Maju Jaya', user: 'System', created_at: '2025-12-04T09:30:00Z' },
    { id: uuid(), field: 'phone', old_value: '+628111000111', new_value: '+628123000001', user: 'Andi Pratama', created_at: '2025-12-03T14:00:00Z' },
    { id: uuid(), field: 'email', old_value: 'andi@gmail.com', new_value: 'andi@example.com', user: 'System', created_at: '2025-12-02T11:00:00Z' },
    { id: uuid(), field: 'city', old_value: 'Bandung', new_value: 'Jakarta', user: 'Admin', created_at: '2025-12-01T08:00:00Z' },
  ];

  // ─── MODEL REGISTRY ─────────────────────────────────────────────
  // Key = model name (as used in component "model" prop).
  // Components call /api/{model}s — so "contact" → /api/contacts
  var modelRegistry = {
    contact: contacts,
    lead: leads,
    event: events,
    task: tasks,
    branch: branches,
    category: categories,
    sale: sales,
    contact_audit: contactAudits,
    kb_card: kbCards,
    kb_comment: kbComments,
    kb_activity: kbActivities,
    kb_user: kbUsers,
  };

  // ─── MATCH URL TO MODEL ─────────────────────────────────────────
  // /api/contacts          → contact  (list)
  // /api/contacts/demo-001 → contact  (read)
  // /api/v1/crm/contacts   → contact  (module-qualified)
  function matchModel(url) {
    // Strip baseUrl and query string
    var path = url.replace(/^https?:\/\/[^/]+/, '').split('?')[0];

    // Pattern: /api/v1/{module}/{model}s or /api/{model}s
    var match = path.match(/\/api\/(?:v\d+\/[^/]+\/)?([a-z_]+)s(?:\/|$)/i);
    if (match) return match[1];
    return null;
  }

  // ─── SIMPLE SEARCH ──────────────────────────────────────────────
  function searchRecords(records, query) {
    if (!query) return records;
    var q = query.toLowerCase();
    return records.filter(function (r) {
      return Object.keys(r).some(function (k) {
        var v = r[k];
        if (typeof v === 'string') return v.toLowerCase().indexOf(q) >= 0;
        if (typeof v === 'number') return String(v).indexOf(q) >= 0;
        return false;
      });
    });
  }

  // ─── SORT ────────────────────────────────────────────────────────
  function sortRecords(records, sort, order) {
    if (!sort) return records;
    var dir = (order || 'asc').toLowerCase() === 'desc' ? -1 : 1;
    return records.slice().sort(function (a, b) {
      var va = a[sort], vb = b[sort];
      if (va == null) return -dir;
      if (vb == null) return dir;
      if (typeof va === 'number' && typeof vb === 'number') return (va - vb) * dir;
      return String(va).localeCompare(String(vb)) * dir;
    });
  }

  // ─── PARSE QUERY PARAMS ─────────────────────────────────────────
  function parseParams(url) {
    var qs = url.split('?')[1] || '';
    var params = {};
    qs.split('&').forEach(function (pair) {
      if (!pair) return;
      var parts = pair.split('=');
      params[decodeURIComponent(parts[0])] = decodeURIComponent(parts[1] || '');
    });
    return params;
  }

  // ─── BUILD LIST RESPONSE ────────────────────────────────────────
  function listResponse(records, params) {
    var page = parseInt(params.page, 10) || 1;
    var pageSize = parseInt(params.page_size, 10) || 20;
    var q = params.q || '';
    var sort = params.sort || '';
    var order = params.order || 'asc';

    var filtered = searchRecords(records, q);
    var sorted = sortRecords(filtered, sort, order);
    var total = sorted.length;
    var totalPages = Math.ceil(total / pageSize) || 1;
    var start = (page - 1) * pageSize;
    var data = sorted.slice(start, start + pageSize);

    return {
      data: data,
      total: total,
      page: page,
      pageSize: pageSize,
      totalPages: totalPages,
    };
  }

  // ─── INTERCEPT FETCH ────────────────────────────────────────────
  var originalFetch = window.fetch;

  window.fetch = function (input, init) {
    var url = typeof input === 'string' ? input : input instanceof Request ? input.url : String(input);

    // Only intercept /api/ calls
    if (url.indexOf('/api/') === -1) {
      return originalFetch.apply(this, arguments);
    }

    var model = matchModel(url);
    var method = (init && init.method) || (input instanceof Request && input.method) || 'GET';

    // POST to /api/upload or /api/uploads — let through (demo has no real files)
    if (url.indexOf('/api/upload') !== -1) {
      return Promise.resolve(new Response(JSON.stringify({ url: '#', id: uuid() }), { status: 200, headers: { 'Content-Type': 'application/json' } }));
    }

    if (!model || !modelRegistry[model]) {
      // Model not in registry — return empty list
      console.warn('[Demo Mock] Unknown model:', model, 'from URL:', url);
      var emptyResp = { data: [], total: 0, page: 1, pageSize: 20, totalPages: 0 };
      return Promise.resolve(new Response(JSON.stringify(emptyResp), { status: 200, headers: { 'Content-Type': 'application/json' } }));
    }

    var records = modelRegistry[model];

    // ─── READ (GET /api/{model}s/{id}) ───────────────────────
    var pathPart = url.replace(/^https?:\/\/[^/]+/, '').split('?')[0];
    var idMatch = pathPart.match(/\/api\/(?:v\d+\/[^/]+\/)?[a-z_]+s\/([a-z0-9-]+)$/i);
    if (method === 'GET' && idMatch) {
      var recordId = idMatch[1];
      var record = records.find(function (r) { return r.id === recordId; }) || records[0];
      return Promise.resolve(new Response(JSON.stringify({ data: record || {} }), { status: 200, headers: { 'Content-Type': 'application/json' } }));
    }

    // ─── CREATE (POST /api/{model}s) ─────────────────────────
    if (method === 'POST' && !idMatch) {
      var body = {};
      if (init && init.body) {
        try { body = JSON.parse(init.body); } catch (e) { /* ignore */ }
      }
      var newRecord = Object.assign({ id: uuid(), created_at: new Date().toISOString() }, body);
      records.push(newRecord);
      return Promise.resolve(new Response(JSON.stringify(newRecord), { status: 201, headers: { 'Content-Type': 'application/json' } }));
    }

    // ─── UPDATE (PUT /api/{model}s/{id}) ─────────────────────
    if ((method === 'PUT' || method === 'PATCH') && idMatch) {
      var updateId = idMatch[1];
      var updateBody = {};
      if (init && init.body) {
        try { updateBody = JSON.parse(init.body); } catch (e) { /* ignore */ }
      }
      var updatedRecord = Object.assign({}, records.find(function (r) { return r.id === updateId; }) || {}, updateBody);
      return Promise.resolve(new Response(JSON.stringify(updatedRecord), { status: 200, headers: { 'Content-Type': 'application/json' } }));
    }

    // ─── DELETE ──────────────────────────────────────────────
    if (method === 'DELETE' && idMatch) {
      return Promise.resolve(new Response(null, { status: 204 }));
    }

    // ─── LIST (GET /api/{model}s) ────────────────────────────
    var params = parseParams(url);
    var resp = listResponse(records, params);
    return Promise.resolve(new Response(JSON.stringify(resp), { status: 200, headers: { 'Content-Type': 'application/json' } }));
  };

  // ─── ALSO INTERCEPT FOR DATATABLE'S dataSource POST ─────────────
  // bc-datatable posts to apiUrl/dataSource with { page, limit, sort, filter }
  // Already covered by fetch interception above.

  console.log('[Demo Mock] API interceptor active. Models:', Object.keys(modelRegistry).join(', '));
})();
