
https://www.postgresql.org/docs/current/config-setting.html#20.1.1.%20Parameter%20Names%20and%20Values

Postgres settings come in one of several variable types:

* enum
* string
* bool
* integer
* real

Some postgres settings are measured in time. They have default units, but postgres can interpret the following units if provided:

* us - microseconds
* ms - milliseconds
* s - seconds
* min - minutes
* h - hours
* d - days

In postgres 14, time-based settings come in three flavors:

* real milliseconds
* integer milliseconds
* integer seconds

How postgres interprets a setting's value depends upon:

1. whether the setting is a real or integer
2. the default units
3. whether units were specified in the input and whether those units are larger, smaller, or equal to the defaults
4. whether the input was a fractional value

statement_timeout is an integer setting with default units of milliseconds.

```
-- no units
set statement_timeout to '155'; show statement_timeout;
┌───────────────────┐
│ statement_timeout │
├───────────────────┤
│ 155ms             │
└───────────────────┘

-- default units
set statement_timeout to '100ms'; show statement_timeout;
┌───────────────────┐
│ statement_timeout │
├───────────────────┤
│ 100ms             │
└───────────────────┘

-- non-default units
set statement_timeout to '100s'; show statement_timeout;
┌───────────────────┐
│ statement_timeout │
├───────────────────┤
│ 100s              │
└───────────────────┘

-- non-default units with fraction
set statement_timeout to '100.3s'; show statement_timeout;
┌───────────────────┐
│ statement_timeout │
├───────────────────┤
│ 100300ms          │
└───────────────────┘

-- default units with fraction
set statement_timeout to '100.7ms'; show statement_timeout;
┌───────────────────┐
│ statement_timeout │
├───────────────────┤
│ 101ms             │
└───────────────────┘

-- no units with fraction
set statement_timeout to '155.3'; show statement_timeout;
┌───────────────────┐
│ statement_timeout │
├───────────────────┤
│ 155ms             │
└───────────────────┘
```

vacuum_cost_delay is a real setting with default units of milliseconds.

```
-- no units
set vacuum_cost_delay to '99'; show vacuum_cost_delay;
┌───────────────────┐
│ vacuum_cost_delay │
├───────────────────┤
│ 99ms              │
└───────────────────┘

-- default units
set vacuum_cost_delay to '100ms'; show vacuum_cost_delay;
┌───────────────────┐
│ vacuum_cost_delay │
├───────────────────┤
│ 100ms             │
└───────────────────┘

-- non-default units
set vacuum_cost_delay to '1000us'; show vacuum_cost_delay;
┌───────────────────┐
│ vacuum_cost_delay │
├───────────────────┤
│ 1ms               │
└───────────────────┘

-- non-default units with fraction
set vacuum_cost_delay to '100.3us'; show vacuum_cost_delay;
┌───────────────────┐
│ vacuum_cost_delay │
├───────────────────┤
│ 100.3us           │
└───────────────────┘

-- default units with fraction
set vacuum_cost_delay to '50.7ms'; show vacuum_cost_delay;
┌───────────────────┐
│ vacuum_cost_delay │
├───────────────────┤
│ 50700us           │
└───────────────────┘

-- no units with fraction
set vacuum_cost_delay to '55.3'; show vacuum_cost_delay;
┌───────────────────┐
│ vacuum_cost_delay │
├───────────────────┤
│ 55300us           │
└───────────────────┘
```
