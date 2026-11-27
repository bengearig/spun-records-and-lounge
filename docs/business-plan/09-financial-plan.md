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

### Loans and SBA programs

The owner has not chosen any funding sources. This section records what research reports about each kind of source, without saying that any of them is available to Spun.

> **Research (unverified):** Sources describe banks as the strictest lenders for a startup with no revenue history, asking for a personal credit score around 670, minimums for revenue and time in business, and collateral. A brand-new retail business therefore usually will not qualify for a conventional bank term loan. Online lenders are described as more flexible and as costing more. A bank line of credit may be easier to get.
>
> Sources: [Lending Tree: startup business loan](https://www.lendingtree.com/business/how-to-get-a-startup-business-loan/), [NerdWallet: business loan requirements](https://www.nerdwallet.com/business/loans/learn/how-to-qualify-for-small-business-loans)
>
> **Verify with:** Lenders directly and the Tennessee Small Business Development Center.

> **Research (unverified):** SBA equity injection requirements are reported as set by the lender, with 10 percent described as standard for riskier deals such as startups. The SBA 7(a) program is described as available to startups. SBA Microloans are reported as up to $50,000 for working capital, inventory, furniture, fixtures, and equipment, made through nonprofit intermediaries. No current Tennessee-specific equity rule or loan limit was found, and some of the sources are lender blogs.
>
> Sources: [SBA: microloan program](https://www.sba.gov/partners/lenders/microloan-program), [SBA: list of microlenders](https://sba.gov/partners/lenders/microloan-program/list-lenders), [Kiplinger: SBA loan rules 2026](https://www.kiplinger.com/business/small-business-loans/sba-loan-rules), [SBA7a.loans: Tennessee](https://www.sba7a.loans/guides/tennessee/)
>
> **Verify with:** The SBA, a Tennessee SBDC counselor, and a microlender.

### Local lenders and free help

> **Research (unverified):** Pathway Lending, a community development financial institution (CDFI), is reported as offering small-business loans and microloans, including an "865 Opportunity Fund" announced in 2023 for $5,000 to $50,000 capital needs in the Knoxville area. Three Roots Capital and the Knoxville Area Urban League are also reported as local CDFIs. LiftFund and People Incorporated Financial Services appear in the SBA microlender list. These programs may have changed since 2023.
>
> Sources: [SCORE Greater Knoxville: business loan options](https://greaterknoxville.score.org/resource/business-loan-options), [Knoxville Chamber: 865 Opportunity Fund (PDF)](https://www.knoxvillechamber.com/wp-content/uploads/2023/01/865OpportunityFund_1sheet_1.19.23.pdf), [SBA: list of microlenders](https://sba.gov/partners/lenders/microloan-program/list-lenders)
>
> **Verify with:** Pathway Lending, each lender directly, and the current SBA list.

> **Research (unverified):** The Tennessee Small Business Development Center, SCORE Greater Knoxville, the Knoxville Entrepreneur Center, and the East Tennessee Development District (which has a revolving loan fund) are reported as free or low-cost help. No current Knoxville startup grant was found, and the state says few federal grants go directly to individual businesses.
>
> Sources: [Knoxville Chamber: small business](https://www.knoxvillechamber.com/regional-enhancement/small-business/), [SCORE Greater Knoxville: business loan options](https://greaterknoxville.score.org/resource/business-loan-options), [Tennessee ECD: Fund Tennessee (SSBCI)](https://www.tn.gov/ecd/small-business/ssbci.html)
>
> **Verify with:** The Knoxville Entrepreneur Center, the Tennessee SBDC Knoxville office, and the East Tennessee Development District.

### Crowdfunding and investors

> **Research (unverified):** Regulation Crowdfunding lets a company raise up to $5,000,000 in 12 months through a funding portal, in exchange for securities. It sets investor limits for non-accredited investors and a one-year resale restriction, and it involves securities-law obligations.
>
> Sources: [SEC: Regulation Crowdfunding](https://www.sec.gov/smallbusiness/exemptofferings/regcrowdfunding)
>
> **Verify with:** The SEC and an attorney.

> **Research (unverified):** Rewards-based platforms (Kickstarter, Indiegogo) and Kiva's zero-interest loans were not covered by the research. The description here is general knowledge and is unverified: on a rewards-based platform, backers pre-buy rewards and no ownership is given up; Kiva is a nonprofit that makes zero-interest loans through local partners, with eligibility requirements.
>
> Sources: none (general knowledge)
>
> **Verify with:** Each platform's own terms.

The owner has said ownership may change as startup capital is raised (see [](08-organization.md#founder) and [](06-licensing-and-legal.md#business-entity)), so any investor or securities offering would change who owns the business.

### Other sources

Personal savings, loans from family and friends, home equity, financing from the landlord or vendors, equipment financing, and gifts were not researched; does the owner want any of these considered?

### Matching sources to the goals

> **Suggestion (to confirm):** Each level in order might fit a different kind of source. The base goal might combine the owner's own contribution with a microloan or other debt. The stretch goals might be funded as cash flow, a later loan, or a campaign allows. The reserve months are part of every goal.
>
> **Confirm with:** The owner.

### Open questions

- How much will the owner contribute, as an amount or none?
- Which funding sources will the owner accept or rule out?
- What terms would the owner accept: rate, repayment period, collateral, and personal guarantee?
- How will the funds be used at each level? The cost data holds the uses.
- Can the owner's credit situation be stated in the plan?
- Ask an attorney before any securities offering, and about how new owners would change the business entity (see [](06-licensing-and-legal.md#business-entity)).

TODO: The owner's own contribution, chosen funding sources, and their terms are not yet stated. Add them here.
