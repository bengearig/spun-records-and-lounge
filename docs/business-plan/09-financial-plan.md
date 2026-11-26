# Financial Plan

## Startup Costs

The one-time costs to open are listed by category and grouped into funding levels: a base that can be funded first and stretch goals added as funding allows, plus a cash reserve of several months of operating costs. Every item and its source is in [](11-appendices.md#startup-cost-detail).

```costs
categories
```

> **Research (unverified):** These amounts come from the cost data file. Its figures are placeholders drawn from unverified research and the owner's working assumptions about floor area, seats, and display units. They are not quotes. Items with no researched figure are shown as zero and flagged as owner input, so every total understates the real cost.
>
> Sources: the cost data file, whose items each name their research; see [](11-appendices.md#startup-cost-detail).
>
> **Verify with:** Supplier quotes, a local general contractor bid, a commercial real estate broker, and an insurance broker.

### What each level adds

Each level builds on the one before it, so the base is funded first and each stretch goal adds to it.

```costs
adds
```

### Base: 1,500 sq ft

The base covers the first 1,500 square feet, with its build-out, furniture, opening inventory, licensing, launch costs, and rent.

```costs
level id=base
```

### Stretch 1: grow to 2,000 sq ft

This level adds more floor area and its build-out, furniture, and rent.

```costs
level id=stretch-1
```

### Stretch 2: grow to 2,500 sq ft

This level adds a further block of floor area and its build-out, furniture, and rent.

```costs
level id=stretch-2
```

### Stretch 3: live music and wall mural

This level adds the live-music items and the wall mural.

```costs
level id=stretch-3
```

### Cash reserve

The reserve is months of operating costs held back, with 3 months at the base and 3 more at each stretch level.

```costs
reserve
```

### Open questions

- The site and its square footage.
- The landlord's build-out allowance and a contractor bid.
- The opening inventory budget and the split of new and used records.
- Seats and display units.
- Equipment the owner already has.
- Insurance quotes.
- Music licensing quotes.
- The number of months of reserve.
- Whether the hemp license fees apply; see [](06-licensing-and-legal.md#hemp-drink-licensing).

## Revenue Model

TODO: Describe how each revenue stream earns income and the assumptions behind it.

## Projections

TODO: Describe multi-year income, expense, and cash flow forecasts.

TODO: Staffing costs are inputs still to come, sized once the [](08-organization.md#hiring-plan) is settled and the [](06-licensing-and-legal.md#employer-obligations) are confirmed:

- Owner pay
- Wages for counter staff, a manager, and an events coordinator
- Employer taxes
- Workers' compensation, if the employee count reaches the threshold

## Break-Even

TODO: Describe the sales level at which revenue covers costs and how it is calculated.

$$\text{BE} = \frac{F}{p - v}$$

Where:

- $F$ is fixed costs
- $p$ is price per unit
- $v$ is variable cost per unit

## Funding

### Funding goals

The plan is staged. The owner intends to fund a base version of the business as a foundation first and to fund the extras as it becomes possible, so each goal below is cumulative. See [](09-financial-plan.md#startup-costs) for what each level contains.

```costs
goals
```

- **Base: 1,500 sq ft.** The goal to open the base is {{costs:goal.base}}.
- **Stretch 1: grow to 2,000 sq ft.** This adds {{costs:level.stretch-1}}, for a goal of {{costs:goal.stretch-1}}.
- **Stretch 2: grow to 2,500 sq ft.** This adds {{costs:level.stretch-2}}, for a goal of {{costs:goal.stretch-2}}.
- **Stretch 3: live music and wall mural.** This adds {{costs:level.stretch-3}}, for a goal of {{costs:goal.stretch-3}}.

These amounts are estimates; see the notice under [](09-financial-plan.md#startup-costs).

TODO: Describe sources of capital, terms, and use of funds.
