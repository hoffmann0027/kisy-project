-- The 10-level hierarchy is a fixed part of the domain model, not sample
-- data, so it ships as a migration rather than a database/seeds script.
INSERT INTO roles (id, level, name, description) VALUES
    (1,  1,  'CEO',                'Unrestricted access to every resource and administrative action.'),
    (2,  2,  'Executive',          'Senior leadership clearance.'),
    (3,  3,  'Director',           'Departmental leadership clearance.'),
    (4,  4,  'Senior Manager',     'Cross-team management clearance.'),
    (5,  5,  'Manager',            'Team management clearance.'),
    (6,  6,  'Team Lead',          'Small-team leadership clearance.'),
    (7,  7,  'Senior Employee',    'Experienced staff clearance.'),
    (8,  8,  'Employee',           'Standard staff clearance.'),
    (9,  9,  'Contractor',         'External limited-access clearance.'),
    (10, 10, 'Guest',              'Minimum clearance.');
