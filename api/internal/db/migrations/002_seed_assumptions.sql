-- Default assumptions per PRD §4.3 / §9 / §11.
-- These are user-editable from the Assumptions screen (NFR-06).

INSERT INTO assumptions (key, category, label, value, unit, description) VALUES
  ('site.lat',                       'site',     'Site latitude',                       '-33.8688'::jsonb, '°',    'NSW default — Sydney'),
  ('site.lon',                       'site',     'Site longitude',                      '151.2093'::jsonb, '°',    'NSW default — Sydney'),
  ('site.tz',                        'site',     'Site timezone',                       '"Australia/Sydney"'::jsonb, 'IANA', NULL),
  ('site.azimuth_deg',               'site',     'Roof azimuth (0=N, 180=S)',           '0'::jsonb,        '°',    'Northerly roof'),
  ('site.tilt_deg',                  'site',     'Roof tilt',                           '22'::jsonb,       '°',    'Typical NSW roof pitch'),

  ('usage.gas_hot_water_fraction',   'usage',    'Gas % attributed to hot water',       '0.80'::jsonb,     '0..1', 'PRD default 80/20 split'),
  ('usage.annual_km',                'usage',    'Annual vehicle kilometres',           '8000'::jsonb,     'km/yr','Current CX5 usage'),
  ('usage.daily_hot_water_l',        'usage',    'Daily hot water demand',              '200'::jsonb,      'L/day','2 occupants in 4-bed sizing'),

  ('panel.standard',                 'panel',    'Standard panel (Trina-class)',        '{"watt":400,"eff":0.205,"temp_coef_per_c":-0.0034,"length_m":1.722,"width_m":1.134}'::jsonb, NULL, NULL),
  ('panel.premium',                  'panel',    'Premium panel (AIKO-class)',          '{"watt":445,"eff":0.227,"temp_coef_per_c":-0.0026,"length_m":1.722,"width_m":1.134}'::jsonb, NULL, NULL),

  ('cost.hphws_gross_aud',           'cost',     'HPHWS gross install',                 '3200'::jsonb,     'AUD',  '250–315 L heat pump HW'),
  ('rebate.hphws_aud',               'rebate',   'HPHWS rebate',                        '1700'::jsonb,     'AUD',  'NSW ESS + STC combined'),
  ('cost.induction_gross_aud',       'cost',     'Induction cooktop install',           '1300'::jsonb,     'AUD',  'Supply + install + gas disconnect'),
  ('rebate.induction_aud',           'rebate',   'Induction rebate',                    '0'::jsonb,        'AUD',  'No current rebate'),
  ('cost.solar_upgrade_gross_aud',   'cost',     'Solar array upgrade',                 '5000'::jsonb,     'AUD',  '14–15 panels supplied + installed'),
  ('rebate.solar_upgrade_aud',       'rebate',   'Solar STCs',                          '1500'::jsonb,     'AUD',  'Approx STC value, depends on deeming'),
  ('cost.ev_gross_aud',              'cost',     'EV mid-range SUV',                    '55000'::jsonb,    'AUD',  'BYD Atto 3 / Ioniq 5 class'),
  ('rebate.ev_aud',                  'rebate',   'EV rebate',                           '0'::jsonb,        'AUD',  'NSW EV incentive phased out 2024'),
  ('cost.petrol_aud_per_l',          'cost',     'Petrol price',                        '1.95'::jsonb,     'AUD/L', NULL),
  ('cost.cx5_l_per_100km',           'cost',     'CX5 fuel economy',                    '8.2'::jsonb,      'L/100km', NULL),

  ('emission.grid_kg_per_kwh',       'emission', 'NSW grid emissions intensity',        '0.79'::jsonb,     'kgCO2e/kWh', 'AEMO/Aus Govt — verify'),
  ('emission.gas_kg_per_kwh_th',     'emission', 'Natural gas emissions',               '0.186'::jsonb,    'kgCO2e/kWh_th', 'NGER 51.53 kg/GJ'),
  ('emission.petrol_kg_per_l',       'emission', 'Petrol emissions',                    '2.31'::jsonb,     'kgCO2e/L', 'NGER'),
  ('emission.panel_kg_each',         'emission', 'Embodied CO2 per panel',              '400'::jsonb,      'kgCO2e/panel', NULL),
  ('emission.ev_embodied_kg',        'emission', 'EV embodied carbon (battery+vehicle delta)', '8500'::jsonb, 'kgCO2e', NULL),
  ('emission.hphws_embodied_kg',     'emission', 'HPHWS embodied carbon',               '450'::jsonb,      'kgCO2e', NULL),
  ('emission.induction_embodied_kg', 'emission', 'Induction cooktop embodied carbon',   '120'::jsonb,      'kgCO2e', NULL),

  ('dispatch.battery_charge_below',  'dispatch', 'Battery charge below price',          '0.05'::jsonb,     'AUD/kWh', NULL),
  ('dispatch.battery_discharge_above','dispatch','Battery discharge above price',       '0.30'::jsonb,     'AUD/kWh', NULL),
  ('dispatch.smart_load_threshold',  'dispatch', 'Smart load price threshold',          '0.10'::jsonb,     'AUD/kWh', 'HPHWS / EV smart charging'),

  ('tariff.supply_aud_per_day',      'cost',     'Daily supply charge',                 '1.10'::jsonb,     'AUD/day', NULL),
  ('tariff.import_cap_aud_per_kwh',  'cost',     'Amber import price cap',              '0.95'::jsonb,     'AUD/kWh', NULL),
  ('tariff.export_floor_aud_per_kwh','cost',     'Amber export floor (negative pays)',  '-0.05'::jsonb,    'AUD/kWh', NULL),

  ('scenario.discount_rate',         'cost',     'Discount rate (display only)',        '0'::jsonb,        '0..1',    'Simple payback per PRD'),
  ('scenario.horizon_years',         'cost',     'Comparison horizon',                  '20'::jsonb,       'years',   NULL)
ON CONFLICT (key) DO NOTHING;
