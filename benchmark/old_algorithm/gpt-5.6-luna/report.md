---

## Research Summary

**Duration:** 661.3s | **Rounds:** 8 | **Queries:** 23 | **URLs analyzed:** 69 | **Model:** openai/gpt-5.6-luna | **Category:** Comparison

---

# Is a Home Battery Worthwhile Without Solar Panels for a UK Home?

## Executive summary

For the household described—an English home using approximately 9,000 kWh of electricity annually, with high daytime home-office demand and limited ability to shift consumption—a grid-charged battery is **unlikely to be financially worthwhile at normal installed prices as of 14 July 2026**.

A battery without solar can still operate profitably: it charges from the grid when electricity is cheap and discharges when electricity is expensive. However, the saving is not the peak-price rate itself. It is the difference between the expensive import price and the effective cost of charging, after allowing for round-trip losses, unused capacity, power limits, degradation, installation cost and replacement risk.

Under the central illustrative assumptions used in this report:

| Option | Indicative first-year electricity cost | Saving versus current tariff | Ten-year result versus competitive flat tariff |
|---|---:|---:|---:|
| 1. Competitive flat tariff, no battery | £2,257 | £167/year | Baseline |
| 2. Time-of-use tariff, no battery | £2,243 | £180/year | About £135 better |
| 3. Time-of-use tariff + 5 kWh battery | £1,991, excluding capital cost | £432/year | About £2,060 worse |
| 4. Time-of-use tariff + 10 kWh battery | £1,915, excluding capital cost | £508/year | About £3,865 worse |

The battery-specific savings are approximately:

- **5 kWh battery:** £252 in the first year;
- **10 kWh battery:** £328 in the first year.

Against assumed installed prices of £4,500 and £7,000 respectively, simple payback is approximately:

- **5 kWh:** 17.9 years;
- **10 kWh:** 21.3 years.

Those simple paybacks are optimistic because they exclude maintenance, financing, inverter replacement, software fees and tariff changes. After allowing for 2% annual degradation over ten years, neither system repays its capital cost in the central scenario.

The strongest recommendation is therefore:

1. Obtain at least 12 months of half-hourly consumption data;
2. Compare a competitive flat tariff with a genuinely suitable time-of-use tariff;
3. Switch tariff first if the figures work; and
4. Consider a battery only if an installer provides a low, fully itemised quotation and a simulation using the household’s actual data.

The **5 kWh battery is the better battery option**, because its capacity is more likely to be used regularly. The 10 kWh battery is unlikely to justify its additional cost unless the home has consistently high evening or 4pm–7pm demand, sufficient charging energy, and a particularly favourable tariff spread.

## Scope and assumptions

This assessment concerns a home in England as at **14 July 2026**. The household:

- Uses approximately 9,000 kWh of electricity annually;
- Pays approximately 24.7p/kWh on a flat rate;
- Uses gas for heating;
- Has a smart meter;
- Can change tariff;
- Uses approximately 10–20 kWh per day in a home office;
- Consumes much of that office electricity during daytime working hours;
- Can shift only a small amount of demand manually;
- Owns the property and expects to remain there for at least ten years;
- Values financial savings more highly than backup power; and
- Does not expect to install solar during the assessment period.

The analysis excludes the value of backup power, because that benefit is highly personal. It also assumes that any battery is used for grid arbitrage rather than export income. The Feed-in Tariff scheme closed to new applicants on 1 April 2019, so a new standalone battery should not be treated as eligible for FIT payments [S55]. Export or flexibility income is also excluded unless a supplier confirms eligibility in writing.

All long-term calculations use constant real prices for illustration. Actual future tariffs may be higher or lower, and the availability of a particular tariff cannot be guaranteed for ten years.

## How the battery would make money

A battery without solar has only one main financial mechanism: **buying electricity at one price and using it later at a higher price**.

Suppose the battery charges at 10p/kWh during an overnight period and has an 88% round-trip efficiency. To deliver 1 kWh later, it needs approximately:

\[
\frac{1}{0.88}=1.136\text{ kWh}
\]

The effective charging cost of that delivered kilowatt-hour is therefore about 11.36p. If the battery discharges during a 30p/kWh peak period, the gross saving is:

\[
30p-11.36p=18.64p/kWh
\]

That 18.64p is the economically relevant margin—not 30p. It must then cover the battery’s capital cost, degradation, standby consumption, maintenance and eventual replacement.

The battery also needs a suitable load to serve. If the home is consuming electricity directly overnight, that consumption may be cheaper to supply directly rather than routing it through the battery. Equally, if much of the home’s demand occurs during daytime hours when the battery is not scheduled to discharge, a large battery may sit partly unused.

This is particularly important here. The home office consumes 10–20 kWh per day, but much of it is used during working hours. A battery charged overnight may be able to support some daytime consumption, but only if:

- It has enough stored energy;
- The control system allows daytime discharge;
- The daytime import price is high enough to justify discharge;
- The battery’s inverter can provide the required power; and
- Discharging during the day does not leave the battery empty before a more expensive evening period.

Time-of-use tariffs can produce savings without a battery where households move demand into cheap periods. Ofgem and government material recognise that flexible consumption can reduce exposure to peak pricing [S53][S54]. Government consultation material has cited savings of more than £200 per year for some households that avoid peak periods, but that figure is conditional and should not be applied automatically to this household [S56][S58].

## Central tariff and battery assumptions

The current flat-rate electricity component is:

\[
9,000 \times £0.247 = £2,223
\]

For the central model, an additional £200 annual standing charge is assumed, giving a total of:

\[
£2,223+£200=\mathbf{£2,423}
\]

The July–September 2026 price-cap benchmark is approximately 26.11p/kWh and 57.19p per day for electricity, implying about £2,559 annually at 9,000 kWh including the standing charge [S40]. The cap is only a benchmark: a competitive tariff may be cheaper.

The report uses the following illustrative alternatives:

| Assumption | Central value |
|---|---:|
| Current unit rate | 24.7p/kWh |
| Competitive flat unit rate | 22.85p/kWh |
| Annual standing charge | £200 |
| TOU overnight rate | 10p/kWh |
| TOU peak rate | 30p/kWh |
| TOU daytime/shoulder rate | 24p/kWh |
| Overnight consumption share | 20% |
| Peak consumption share | 25% |
| Daytime/shoulder share | 55% |
| Round-trip battery efficiency | 88% |
| Battery degradation | 2% annually |
| 5 kWh installed price | £4,500 |
| 10 kWh installed price | £7,000 |

The competitive flat rate is an illustrative postcode-specific benchmark, not a guaranteed market offer. Some market pages cite electricity rates beginning around 22.85p/kWh, while regional fixed-rate examples around 27.78p/kWh would be more expensive than the assumed current rate [S3]. A proper decision must use an actual quotation.

The TOU schedule is also illustrative:

- 10p/kWh overnight: 20% of consumption;
- 30p/kWh peak: 25%;
- 24p/kWh daytime or shoulder: 55%.

Its weighted average rate is:

\[
(20\%\times10p)+(25\%\times30p)+(55\%\times24p)=22.7p/kWh
\]

Some suppliers advertise overnight rates around 6.49–6.99p/kWh, but these offers may be restricted to electric-vehicle customers or subject to other eligibility conditions [S45][S46][S49]. Octopus Go, for example, is specifically associated with home EV charging, while certain battery-focused tariffs require solar panels, export arrangements or approved equipment [S20][S21][S22][S24]. A headline overnight price should not be assumed to be available for a standalone home battery.

## Comparison Table

| Criterion | Option 1: Competitive flat tariff, no battery | Option 2: TOU tariff, no battery | Option 3: TOU + 5 kWh battery | Option 4: TOU + 10 kWh battery |
|---|---|---|---|---|
| Upfront cost | None | None | Approximately £4,500 | Approximately £7,000 |
| Indicative first-year bill | £2,257 | £2,243 | £1,991 | £1,915 |
| First-year saving versus current tariff | £167 | £180 | £432 | £508 |
| Battery-specific saving | None | None | £252 | £328 |
| Simple payback | Immediate tariff saving | Immediate tariff saving | About 17.9 years | About 21.3 years |
| Ten-year capital recovery | Not applicable | Not applicable | No, in central case | No, in central case |
| Dependence on tariff spread | Low | Medium | High | Very high |
| Dependence on load shifting | Low | Medium | High | Very high |
| Effect of high daytime demand | Generally neutral | Could help if daytime is cheap | Can reduce battery cycling | Can leave capacity underused |
| Round-trip losses | None | None | Approximately 12% assumed | Approximately 12% assumed |
| Degradation risk | None | None | Yes | Yes |
| Inverter/replacement risk | None | None | Yes | Yes |
| Backup capability | No | No | Possible, at extra cost | Possible, at extra cost |
| Financial risk | Lowest | Low | Medium/high | High |
| Best use case | Reliable low-cost electricity | Demand already matches cheap periods | Smaller, frequently used arbitrage system | High evening demand and unusually favourable pricing |
| Overall central-case rating | **Best low-risk option** | **Worth testing** | **Best battery option, but weak return** | **Least attractive financially** |

## Option 1: Competitive flat-rate tariff without a battery

### Strengths

This is the lowest-risk option. It requires no capital expenditure, no installation, no battery warranty assessment and no dependence on a particular control platform.

At 22.85p/kWh plus the assumed £200 standing charge, annual cost would be approximately:

\[
9,000\times£0.2285+£200=\mathbf{£2,256.50}
\]

Rounded, that is £2,257 per year—approximately £167 below the assumed current cost of £2,423.

The saving is available regardless of when the household consumes electricity. That matters because the home office operates during the day and the household has limited ability to shift demand manually. A flat tariff avoids the risk that unavoidable daytime or evening consumption falls into a punitive peak band.

### Weaknesses

A competitive flat tariff does not exploit unusually cheap overnight or dynamic prices. If the household were able to use a very large share of electricity during a cheap period, a TOU tariff might save more.

There is also tariff-renewal risk. A fixed or discounted rate may expire, and the next available tariff may be less attractive. The homeowner should compare the actual standing charge as well as the unit rate. At 9,000 kWh annually, a 1p/kWh difference is worth approximately £90 per year, so small rate changes matter.

### Ideal use case

This is the ideal option for a household that wants savings without operational complexity, especially where consumption is concentrated in periods that are not cheap under the available TOU tariff.

For this household, it is the appropriate financial benchmark against which every battery quotation should be compared.

## Option 2: Time-of-use tariff without a battery

### Strengths

A TOU tariff provides the benefits of time-based pricing without battery capital cost. If electricity is already consumed during cheap periods, the household can save without suffering battery losses.

This may be particularly relevant to the office. Some tariffs have cheap daytime periods rather than only overnight windows. EDF’s proposed three-rate structure, for example, describes a cheaper daytime “amber” period from approximately 6am to 4pm, which could suit daytime office use better than an overnight-only tariff [S42]. That kind of tariff may be more valuable to this household than a tariff offering cheap electricity only between 11pm and 6am.

A tariff-only switch is also reversible. If the tariff proves unsuitable, the homeowner can change again without having to recover thousands of pounds of sunk battery cost.

### Weaknesses

A TOU tariff is not automatically cheaper. Under the central assumptions, the weighted average rate is 22.7p/kWh, only slightly below the assumed 22.85p/kWh flat tariff. The annual saving compared with the flat option is only approximately £13.50.

If the household has a large amount of unavoidable peak consumption, a TOU tariff can be more expensive than a flat rate. One source explicitly warns that smart tariffs may cost more when households continue to use substantial electricity in expensive periods [S2]. Government flexibility material makes the same basic point: savings depend on actual ability to alter consumption [S53][S54].

### Ideal use case

This is best for a household whose half-hourly demand naturally overlaps with cheap tariff periods, or which can shift appliances, hot-water heating, vehicle charging or other loads automatically.

For the present household, a TOU tariff is worth testing before buying a battery—but the correct tariff must include a cheap daytime period or have a sufficiently large overnight-to-peak spread.

## Option 3: Approximately 5 kWh battery plus a TOU tariff

### Strengths

The 5 kWh battery is the more credible battery size. It has approximately 4.5 kWh of usable capacity under the central assumption, rather than the full nominal 5 kWh. It is small enough that the household may be able to charge and discharge it regularly without requiring an unsupported full cycle every day.

The model assumes 300 useful equivalent cycles in the first year:

\[
4.5\text{ kWh}\times300=1,350\text{ kWh delivered}
\]

At an effective arbitrage margin of 18.64p/kWh, that produces approximately:

\[
1,350\times£0.1864=\mathbf{£252}
\]

of first-year battery-specific saving.

The 5 kWh option is also less exposed to under-utilisation. If the home has only 4–5 kWh of suitable peak-period demand or available overnight charging energy, a 10 kWh battery would have unused capacity that generates no return.

### Weaknesses

At an assumed installed price of £4,500, the simple payback is:

\[
£4,500/£252=\mathbf{17.9\ years}
\]

That is longer than the household’s ten-year horizon and may exceed the useful economic life of the inverter or battery. It also ignores maintenance, software costs, financing and replacement.

The home office’s daytime demand is a mixed factor. It increases total usage, but it does not necessarily increase profitable battery cycling. If the office consumes electricity in a daytime band priced at 24p/kWh, discharging a battery charged at 10p/kWh may save less than discharging during a 30p/kWh peak period. The system may also be empty by the time the evening peak arrives.

### Ideal use case

A 5 kWh battery is appropriate only where:

- A battery-compatible tariff permits grid charging;
- The off-peak-to-peak spread is large and persistent;
- The home has reliable peak-period demand;
- The battery can discharge at the required power;
- The installed quotation is substantially below the central assumption; or
- Backup power has significant personal value.

It is the best battery option, but not the best financial option in the central case.

## Option 4: Approximately 10 kWh battery plus a TOU tariff

### Strengths

The 10 kWh battery can theoretically deliver more energy during expensive periods. Assuming 8 kWh usable capacity and 220 equivalent cycles annually:

\[
8\text{ kWh}\times220=1,760\text{ kWh delivered}
\]

At the central arbitrage margin, this is approximately £328 of first-year battery saving.

A larger battery may be useful where the home has substantial evening demand, multiple occupants, electric cooking, electric hot water or other loads continuing through the 4pm–7pm peak.

### Weaknesses

The additional capacity produces only about £76 more annual battery saving than the 5 kWh system in the central model:

\[
£328-£252=\mathbf{£76}
\]

Yet the assumed installed price is £2,500 higher. That makes the incremental economics particularly poor.

The 10 kWh battery also needs more energy to fill. At 88% efficiency, delivering 8 kWh requires approximately:

\[
8/0.88=9.09\text{ kWh}
\]

of charging electricity. If 20% of the household’s annual consumption occurs overnight, overnight use is approximately 4.9 kWh per day. Some of that is consumed directly, leaving less than 9 kWh available to fill a 10 kWh system on many days.

The battery may also be power-limited. Capacity is measured in kWh, but the inverter’s ability to deliver power is measured in kW. A 10 kWh battery with a 2.5 kW inverter cannot supply a simultaneous 4–5 kW household load merely because it contains 10 kWh of energy.

### Ideal use case

The 10 kWh system is suitable only where half-hourly data demonstrates sustained high peak-period demand and the battery can be cycled deeply and regularly. It may also be chosen for backup, but backup capacity is a separate benefit and should not be allowed to disguise poor arbitrage economics.

For this household, it is unlikely to be the financially optimal size.

## Annual electricity costs and savings

### Central case: 9,000 kWh annually

| Option | Calculation | Annual cost |
|---|---|---:|
| Current tariff | 9,000 × 24.7p + £200 | £2,423 |
| Competitive flat tariff | 9,000 × 22.85p + £200 | £2,257 |
| TOU, no battery | 9,000 × 22.7p + £200 | £2,243 |
| TOU + 5 kWh battery | £2,243 − £252 | £1,991 |
| TOU + 10 kWh battery | £2,243 − £328 | £1,915 |

The battery-inclusive figures are **energy-bill costs only**. They do not deduct the battery purchase price.

The 10 kWh battery appears to produce the lowest annual bill, but that is not the same as producing the best financial outcome. The homeowner pays approximately £2,500 more for only around £76 of additional annual saving in the central case.

## Simple payback and ten-year outcome

### Simple payback

| Battery | Installed cost | First-year saving | Simple payback |
|---|---:|---:|---:|
| 5 kWh | £4,500 | £252 | 17.9 years |
| 10 kWh | £7,000 | £328 | 21.3 years |

For a battery to repay itself within ten years before allowing for degradation, its maximum cost would be approximately:

- 5 kWh: £2,520;
- 10 kWh: £3,280.

A robust ten-year investment case would require an even lower price, because output declines and equipment may incur service or replacement costs.

### Ten-year outcome with 2% degradation

At 2% annual degradation, the ten-year output is approximately 91.5% of ten times first-year output.

Approximate ten-year battery savings are therefore:

- 5 kWh: £2,306;
- 10 kWh: £3,001.

| Option | Ten-year electricity cost | Capital cost | Ten-year total |
|---|---:|---:|---:|
| Competitive flat tariff | £22,565 | £0 | £22,565 |
| TOU, no battery | £22,430 | £0 | £22,430 |
| TOU + 5 kWh battery | Approximately £20,124 | £4,500 | £24,624 |
| TOU + 10 kWh battery | Approximately £19,429 | £7,000 | £26,429 |

Compared with the competitive flat tariff:

- The TOU-only option is approximately £135 better;
- The 5 kWh battery is approximately £2,059 worse;
- The 10 kWh battery is approximately £3,864 worse.

These results are not predictions. They are an illustration of how a battery can reduce the annual bill while still losing money overall.

## Sensitivity analysis

### Annual consumption

The battery does not benefit equally from every additional kilowatt-hour. If additional electricity is consumed during cheap daytime hours, it may not increase profitable battery discharge. If it is consumed during the expensive peak period, it can improve battery economics.

Using the model’s utilisation assumptions:

| Annual usage | 5 kWh delivered energy | 10 kWh delivered energy |
|---:|---:|---:|
| 6,000 kWh | 900 kWh | 1,120 kWh |
| 9,000 kWh | 1,350 kWh | 1,760 kWh |
| 11,000 kWh | 1,485 kWh | 2,000 kWh |

Approximate first-year costs are:

| Annual use | Competitive flat | TOU only | TOU + 5 kWh | TOU + 10 kWh |
|---:|---:|---:|---:|---:|
| 6,000 kWh | £1,571 | £1,562 | £1,394 | £1,353 |
| 9,000 kWh | £2,257 | £2,243 | £1,991 | £1,915 |
| 11,000 kWh | £2,714 | £2,697 | £2,421 | £2,324 |

At 6,000 kWh, the batteries are less likely to cycle sufficiently and their ten-year economics worsen. At 11,000 kWh, the case improves only if the extra demand occurs in the battery’s profitable discharge period.

### Tariff-price spread

The spread is the single most important operating variable.

| Off-peak | Peak | Approximate saving per delivered kWh | 5 kWh annual value | 10 kWh annual value |
|---:|---:|---:|---:|---:|
| 15p | 25p | 7.95p | £107 | £140 |
| 10p | 30p | 18.64p | £252 | £328 |
| 5p | 35p | 29.32p | £396 | £516 |
| 5p | 45p | 39.32p | £531 | £692 |

At a persistent 5p/35p spread, the 5 kWh battery’s ten-year gross savings after degradation could approach £3,620—still below the assumed £4,500 installed cost. At 5p/45p, its ten-year gross savings could approach £4,860, making payback possible, though still vulnerable to maintenance, replacement and tariff changes.

Such a spread must be available frequently, not merely observed during occasional negative-price or extreme wholesale events. Agile-style tariffs can be highly volatile, and supplier information warns that dynamic pricing may sometimes cost more than a standard tariff [S1][S5][S21].

### Peak-period consumption

The household’s share of peak-period usage determines whether a battery can discharge productively.

Under the illustrative schedule:

| Peak consumption share | Approximate TOU average | Comparison with 22.85p flat rate |
|---:|---:|---:|
| 15% | 22.1p/kWh | About £68/year cheaper |
| 25% | 22.7p/kWh | About £14/year cheaper |
| 35% | 23.3p/kWh | About £41/year more expensive |

At 35% peak consumption, the TOU tariff is already more expensive than the competitive flat tariff before considering a battery. A battery may offset that premium, but only if its discharge window matches the consumption.

The high office usage is therefore not automatically positive. A daytime-heavy profile may reduce evening peak exposure, but it can also make overnight-only tariffs unattractive if the daytime period is expensive.

### Installation prices

Approximate ten-year battery values in the central price-spread case are £2,300 for the 5 kWh system and £3,000 for the 10 kWh system.

| Installed price | 5 kWh outcome | 10 kWh outcome |
|---|---:|---:|
| £2,500 / £4,000 | Approximately £200 better than flat / £1,000 worse | — |
| £3,500 / £5,500 | Approximately £800 worse | Approximately £2,500 worse |
| £4,500 / £7,000 | Approximately £2,100 worse | Approximately £3,900 worse |
| £6,000 / £9,000 | Approximately £3,600 worse | Approximately £5,900 worse |

The 5 kWh battery becomes potentially viable at a very low installed price or with a much more favourable spread. The 10 kWh battery needs both a low cost and high utilisation of its additional capacity.

### Degradation and efficiency

At 4% annual degradation rather than 2%, ten-year output falls to roughly 83% of the undegraded total. Approximate average annual battery savings would fall to:

- 5 kWh: about £209;
- 10 kWh: about £272.

Round-trip efficiency also matters:

| Efficiency | Charging electricity needed per delivered kWh | Effective cost at 10p/kWh |
|---:|---:|---:|
| 88% | 1.14 kWh | 11.4p |
| 80% | 1.25 kWh | 12.5p |
| 70% | 1.43 kWh | 14.3p |

A manufacturer’s headline “capacity” should not be confused with guaranteed usable capacity. The quotation should state usable AC-side energy, continuous and peak charge/discharge power, standby consumption, retained capacity at the end of the warranty, cycle or throughput limits, and whether grid arbitrage is permitted.

## Shared Considerations

### Tariff eligibility and uncertainty

A suitable tariff must explicitly permit the proposed operating pattern. Some cheap overnight tariffs are designed for EVs and may require a vehicle or charger [S20][S49]. Some battery tariffs require solar, an export MPAN or approved equipment [S22][S24][S25]. The homeowner should obtain written confirmation that:

- A standalone battery may charge from the grid;
- The tariff does not prohibit discharge into household loads;
- The battery’s control software is compatible;
- Supplier-controlled charging is acceptable; and
- The tariff does not require solar or an EV.

Tariffs may change, be withdrawn or become less attractive. Dynamic tariffs can provide exceptional spreads but also expose the household to high prices [S1][S5]. A ten-year battery investment should not be based solely on one temporary tariff.

### VAT and incentives

HMRC guidance indicates that qualifying energy-saving-material installations may receive 0% VAT during the relevant period, with the relief currently important before 31 March 2027 [S26]. Installer commentary specifically argues that standalone domestic batteries can qualify without solar [S59], but the homeowner should obtain the installer’s written VAT basis because the treatment depends on the statutory conditions and the precise installation.

No battery should be bought on the assumption that FIT, SEG or flexibility income will automatically be available. SEG is primarily associated with eligible renewable generation, and grid-charged battery exports require supplier confirmation [S38]. The central model therefore assumes zero export income.

### Installation and technical limitations

A battery quotation must distinguish:

- Nominal capacity from usable capacity;
- Energy capacity in kWh from power output in kW;
- Battery efficiency from whole-system efficiency;
- Standard operation from backup operation;
- Battery cost from inverter cost;
- Hardware warranty from software support; and
- Warranty duration from guaranteed retained capacity.

A standard grid-connected battery may not provide power during a power cut. Backup normally requires a gateway, changeover equipment, protected circuits and additional installation work. Backup should be priced separately rather than assumed to be included.

### Maintenance and replacement

The central model assumes no maintenance cost and no replacement. That favours the battery.

Even £100 annually for monitoring or maintenance removes £1,000 from the ten-year result. A £2,000 inverter replacement removes a further £2,000. Battery replacement could be more expensive and may occur before the end of the homeowner’s ten-year period.

The warranty should address grid charging, annual throughput, cycle limits, retained capacity, inverter failure, labour and call-out costs, software availability and replacement arrangements.

### Consumption data

The homeowner should obtain at least 12 months of half-hourly import data. Annual consumption alone is insufficient. The analysis should identify:

- Overnight consumption;
- Daytime office demand;
- 4pm–7pm demand;
- Evening demand;
- Maximum half-hourly import;
- Weekday and weekend differences;
- Seasonal variation;
- Electricity consumed directly overnight;
- Flexible appliances; and
- The amount of demand a battery could physically serve.

The model should simulate battery state of charge, minimum reserve, actual charge/discharge power, round-trip efficiency, standby use, tariff windows, degradation and periods when the battery cannot discharge because demand is too low.

## Conditions under which the conclusion changes

A battery could become financially worthwhile if most of the following conditions apply:

- Off-peak electricity is consistently available at approximately 5–8p/kWh;
- Peak electricity is reliably 35–45p/kWh or higher;
- The spread persists for much of the battery’s useful life;
- The home has substantial demand during the expensive discharge period;
- The 5 kWh battery can complete approximately 300–350 useful cycles annually;
- The 10 kWh battery’s additional capacity is regularly used;
- The inverter can meet simultaneous household demand;
- Installation cost is around £2,500–£3,000 for 5 kWh or materially below £7,000 for 10 kWh;
- Warranty terms allow grid arbitrage;
- Degradation is modest; and
- No major inverter, maintenance or replacement cost is incurred.

A tariff-only switch becomes more attractive if the supplier offers a cheap daytime period matching the office load. EDF’s described daytime amber period illustrates why tariff design matters for this particular household: a daytime discount could be valuable without forcing electricity through a battery and incurring losses [S42].

The case becomes weaker if the household consumes most electricity during expensive periods but cannot shift it, if overnight demand is already high, if the battery requires a large backup reserve, or if the tariff’s cheap hours do not match the home’s operating pattern.

## What the homeowner should obtain before buying

### Consumption information

Request:

- Twelve months of half-hourly smart-meter data;
- Separate weekday and weekend analysis;
- Overnight consumption;
- 4pm–7pm consumption;
- Evening consumption;
- Office load by hour;
- Maximum half-hourly demand;
- Seasonal patterns; and
- Flexible-load opportunities.

### Tariff information

Obtain written details of:

- Every import rate;
- Exact time bands;
- Standing charge;
- VAT;
- Eligibility requirements;
- Whether battery charging is permitted;
- Whether an EV, solar or approved battery is required;
- Supplier-control arrangements;
- Export rules;
- Software or subscription charges;
- Contract length;
- Exit fees; and
- Rate-change mechanisms.

### Battery quotations

Obtain at least three itemised quotations showing:

- Manufacturer and model;
- Nominal and usable capacity;
- Continuous and peak charge/discharge power;
- AC-side round-trip efficiency;
- Minimum state of charge;
- Standby consumption;
- Inverter rating;
- Backup equipment and protected circuits;
- Installation and electrical modifications;
- DNO requirements;
- VAT treatment;
- Warranty duration;
- Guaranteed end-of-warranty capacity;
- Throughput and cycle limits;
- Grid-arbitrage permission;
- Maintenance;
- Software fees;
- Inverter replacement coverage; and
- Battery replacement assumptions.

The installer should provide a written calculation based on the household’s own half-hourly data. Reject any estimate that assumes one full charge-discharge cycle every day without demonstrating that the load profile and tariff windows support it.

## Best For verdicts

**Best for lowest financial risk: Option 1, a competitive flat tariff without a battery.** It provides a predictable saving with no capital expenditure and is least affected by the home office’s daytime-heavy demand.

**Best for testing flexibility: Option 2, a time-of-use tariff without a battery.** It should be tested first, particularly if a tariff offers a cheap daytime period matching office use. Under the central assumptions, its advantage over a competitive flat tariff is modest, but the downside is limited and reversible.

**Best battery option: Option 3, the approximately 5 kWh battery.** It is more likely than a 10 kWh battery to cycle regularly and has a lower capital cost. Nevertheless, it is not financially attractive at an assumed £4,500 installed price unless tariff spreads or installation prices are substantially better than the central case.

**Best for high evening demand or backup: Option 4, the approximately 10 kWh battery.** It may be appropriate where measured 4pm–7pm and evening demand is consistently high, but it is the weakest central-case financial choice because its extra capacity is insufficiently utilised.

## Conclusion

A home battery without solar **can** save money in England by charging from the grid at cheap times and discharging during expensive periods. However, for this particular home, the central evidence points against purchasing one primarily for financial savings.

The unusually high daytime office demand does not guarantee high battery value. Much of that electricity may be consumed outside the most expensive period, and some overnight electricity would be consumed directly rather than stored. Limited manual flexibility further reduces the certainty of frequent profitable cycling.

At approximately £4,500 installed, a 5 kWh battery produces an illustrative simple payback of around 18 years and remains about £2,060 worse than the competitive-flat benchmark over ten years after modest degradation. At approximately £7,000, the 10 kWh battery produces an illustrative simple payback of around 21 years and remains about £3,865 worse over ten years.

Therefore:

- **Neither battery is likely to pay for itself under ordinary central-case assumptions.**
- **The 5 kWh system is preferable to the 10 kWh system if a battery is purchased.**
- **Changing tariff without buying a battery is the sensible first step.**
- **A competitive flat tariff is the strongest low-risk option.**
- **A TOU tariff could be better if its cheap periods match the office’s daytime use or the household’s measured demand.**
- **A battery becomes plausible only with a low installed price, a large and durable price spread, sufficient peak-period demand and favourable warranty terms.**

The strongest argument against this recommendation is that electricity-price spreads may become much larger, and a carefully controlled battery could then earn more than the central model assumes. But that possibility is not a dependable ten-year investment case. The homeowner should buy a battery only after obtaining actual half-hourly data, complete tariff terms and comparable quotations demonstrating that the expected savings—not merely the annual bill reduction—recover the capital cost.

### Sources

- [S1] [Agile Octopus | Half-hourly UK Wholesale Price Tracker | Octopus Energy](https://octopus.energy/agile/)
- [S2] [Smart Meter Tariffs Ranked July 2026: Agile, Go, Cosy & Flux](https://www.energyplus.co.uk/costsavingadvice/smart-meter-tariffs-ranked)
- [S3] [Octopus Energy Tariffs & Rates UK (Updated July 2026)](https://www.energyplus.co.uk/suppliers/octopus-energy)
- [S4] [Smart time of use tariffs: all you need to know - Energy Saving Trust](https://energysavingtrust.org.uk/time-use-tariffs-all-you-need-know/)
- [S5] [Octopus Tracker | Daily UK Wholesale Price Tracker | Octopus Energy](https://octopus.energy/smart/tracker/)
- [S6] [How do I get the energy price cut from April 2026? | Octopus Energy](https://octopus.energy/blog/april-2026-price-cap-updates/)
- [S7] [Understanding Octopus tariffs | Switch to Octopus](https://switchtooctopus.co.uk/docs/tariffs/understanding-tariffs/)
- [S8] [Is it time to fix your energy or stay on the Price Cap? Martin Lewis](https://www.moneysavingexpert.com/utilities/are-there-any-cheap-fixed-energy-deals-currently-worth-it/)
- [S9] [Should I Add a Solar Battery in 2026? UK Guide (July 2026)](https://www.energyplus.co.uk/solar/should-i-add-a-battery-to-my-solar-panels-in-2026)
- [S10] [Solar Battery Storage System for UK Homes | UKEM Group](https://www.ukem.co.uk/solar-battery-storage/)
- [S11] [Octopus Energy: The UK's most awarded energy supplier](https://octopus.energy/)
- [S12] [Best Time-of-Use Tariff UK 2026: Octopus Go vs Agile](https://amprenewables.co.uk/blog/best-time-of-use-tariff-uk-2026)
- [S13] [UK Energy Suppliers Offering Free Electricity Hours Tariffs (July 2026)](https://www.energyplus.co.uk/costsavingadvice/energy-suppliers-offering-free-electricity-hours-tariff-uk)
- [S14] [Energy Supplier, Switch Gas & Electricity Provider | OVO Energy](https://www.ovoenergy.com/)
- [S15] [Best Heat Pump Tariffs in 2026: How to Cut Your Heat Pump Running Costs With Time of Use Electricity | Cucumber Eco](https://cucumbereco.co.uk/blog/heat-pump-tariffs-2026)
- [S16] [What are the best SEG rates? | All 38 tariffs ranked [2026]](https://www.sunsave.energy/solar-panels-advice/exporting-to-the-grid/best-seg-rates)
- [S17] [Martin Lewis: For most people, July’s 13% Energy Price Cap rise is voluntary – it can (and should) be avoided](https://www.moneysavingexpert.com/news/2026/05/martin-lewis-energy-price-cap-rise-july/)
- [S18] [Home Battery Storage Without Solar Panels UK — 2026 Costs](https://www.energyplus.co.uk/solar/home-battery-storage-without-solar-panels-uk)
- [S19] [Solar Export Tariffs in 2026: How to Earn from Your Solar Panels](https://switchtogether.co.uk/resource-hub/blog/solar-export-tariffs)
- [S20] [Octopus Go | Off Peak EV Charging for Any Car & Charger | Octopus Energy](https://octopus.energy/go/)
- [S21] [Smart Tariffs - Terms and Conditions | Octopus Energy](https://octopus.energy/policies/smart-tariffs-terms-and-condition/)
- [S22] [Octopus Flux | Energy Tariff Designed for Solar & Batteries | Octopus Energy](https://octopus.energy/smart/flux/)
- [S23] [EV charger installation with Octopus Energy: FAQs | Octopus Energy](https://octopus.energy/ev-charger-faq/)
- [S24] [Intelligent Octopus Flux | Automated export to support the grid | Octopus Energy](https://octopus.energy/smart/intelligent-octopus-flux/)
- [S25] [Intelligent Octopus Flux: Frequently Asked Questions | Octopus Energy](https://octopus.energy/intelligent-flux-faqs/)
- [S26] [Energy-saving materials and heating equipment (VAT Notice 708/6) - GOV.UK](https://www.gov.uk/guidance/vat-on-energy-saving-materials-and-heating-equipment-notice-7086)
- [S27] [Domestic reverse charge procedure (VAT Notice 735) - GOV.UK](https://www.gov.uk/guidance/the-vat-domestic-reverse-charge-procedure-notice-735)
- [S28] [Warm Homes Plan (HTML) - GOV.UK](https://www.gov.uk/government/publications/warm-homes-plan/warm-homes-plan-html)
- [S29] [The 11 best EV tariffs [UK, 2026]](https://www.sunsave.energy/solar-panels-advice/exporting-to-the-grid/best-ev-charging-tariffs)
- [S30] [Best EV Tariff UK 2026: A Driver's Guide to Cheaper Home Charging | loveelectric](https://www.loveelectric.cars/blog/best-ev-tariff-uk)
- [S31] [Solar Tariffs - Compare and Save on Energy Bills | EDF](https://www.edfenergy.com/solar/solar-tariffs)
- [S32] [Introducing battery-only installation with Octopus | Octopus Energy](https://octopus.energy/blog/battery-only-installation/)
- [S33] [Grid Guru July 2026 Newsletter: Key Updates in UK Solar & Energy Market](https://www.gridguru.co.uk/post/grid-guru-july-2026-newsletter)
- [S34] [Understanding time of use tariffs and energy bills.](https://www.eonnext.com/energy/guides/time-of-use-tariffs)
- [S35] [Best Solar Batteries UK 2026: 7 Home Batteries Ranked by £/kWh | SolarGridCheck.co.uk](https://solargridcheck.co.uk/best-home-batteries-uk)
- [S36] [Britain's first solar tariff with no standing charge launched](https://www.edfenergy.com/media-centre/edf-launches-britains-first-solar-tariff-no-standing-charge)
- [S37] [Change or Renew Your Energy Tariff | New & Existing Customers | EDF](https://www.edfenergy.com/gas-and-electricity/change-energy-tariff)
- [S38] [Smart Export Guarantee & Tariff | Solar Energy | EDF](https://www.edfenergy.com/energy-efficiency/smart-export-tariff)
- [S39] [Choosing the Best Energy Tariffs - How to pick the right one for you | EDF](https://www.edfenergy.com/energywise/choosing-best-energy-tariff)
- [S40] [July 2026 Energy Price Cap to rise by 13% | EDF](https://www.edfenergy.com/energywise/energy-price-cap-july-2026)
- [S41] [Should I fix my energy tariff before the July price cap? | Octopus Energy](https://octopus.energy/blog/should-I-fix-before-july-2026-price-cap/)
- [S42] [EDF launches first-of-its-kind three-rate tariff with free electricity hours | EDF](https://www.edfenergy.com/media-centre/edf-launches-first-its-kind-three-rate-tariff-free-electricity-hours)
- [S43] [Electricity Tariffs: A Guide for Electric Car Drivers | EDF](https://www.edfenergy.com/electric-cars/tariffs-guide)
- [S44] [Improving the energy efficiency of socially rented homes in England: government response  - GOV.UK](https://www.gov.uk/government/consultations/improving-the-energy-efficiency-of-socially-rented-homes-in-england/outcome/improving-the-energy-efficiency-of-socially-rented-homes-in-england-government-response)
- [S45] [EDF's EV Tariffs For Your Car And Home | EV Tariffs | EDF](https://www.edfenergy.com/electric-cars/ev-tariffs)
- [S46] [EDF launches cheapest overnight charging on EV tariffs | EDF](https://www.edfenergy.com/media-centre/edf-launches-cheapest-overnight-charging-ev-tariffs)
- [S47] [Energy Price Cap: Everything You Need To Know | OVO Energy](https://www.ovoenergy.com/pricecap)
- [S48] [Earn free electricity on Sundays with our Sunday Saver Challenge | EDF](https://www.edfenergy.com/energy-efficiency/sunday-saver-challenge)
- [S49] [Overnight EV Tariff For Your Car And Home | GoElectric | EDF](https://www.edfenergy.com/electric-cars/ev-tariffs/goelectric)
- [S50] [Boiler Upgrade Scheme (BUS) - Property owners | Ofgem](https://www.ofgem.gov.uk/environmental-and-social-schemes/boiler-upgrade-scheme-bus/boiler-upgrade-scheme-bus-property-owners)
- [S51] [bakewell-town-guide-2025.pdf](https://www.bakewelltowncouncil.gov.uk/uploads/bakewell-town-guide-2025.pdf?v=1750841577)
- [S52] [The UK's Integrated National Energy and Climate Plan - GOV.UK](https://assets.publishing.service.gov.uk/media/60bdd2d2e90e0743ae8c284e/uk-integrated-national-energy-climate-plan-necp-31-january-2020.pdf)
- [S53] [Clean flexibility roadmap: July 2026 update (accessible webpage) - GOV.UK](https://www.gov.uk/government/publications/clean-flexibility-roadmap/clean-flexibility-roadmap-july-2026-update-accessible-webpage)
- [S54] [Clean flexibility roadmap (accessible webpage) - GOV.UK](https://www.gov.uk/government/publications/clean-flexibility-roadmap/clean-flexibility-roadmap)
- [S55] [Feed-in Tariffs (FIT) | Ofgem](https://www.ofgem.gov.uk/environmental-and-social-schemes/feed-tariffs-fit)
- [S56] [Smart Secure Electricity Systems (SSES) Programme: first phase energy smart appliances regulations - consultation document (accessible webpage) - GOV.UK](https://www.gov.uk/government/consultations/smart-secure-electricity-systems-sses-programme-first-phase-energy-smart-appliances-regulations/smart-secure-electricity-systems-sses-programme-first-phase-energy-smart-appliances-regulations-consultation-document-accessible-webpage)
- [S57] [Assessing the case for community batteries: call for evidence](https://assets.publishing.service.gov.uk/media/6a2041a058aae59498cb29e5/assessing-the-case-for-community-batteries-call-for-evidence.pdf)
- [S58] [Smart Secure Electricity Systems (SSES) Programme: Enduring Governance (accessible webpage) - GOV.UK](https://www.gov.uk/government/consultations/smart-secure-electricity-systems-programme-sses-enduring-governance/smart-secure-electricity-systems-sses-programme-enduring-governance-accessible-webpage)
- [S59] [Zero VAT on Home Battery Storage: The HMRC Rules, Real Savings, and What to Check Before March 2027 | Mackie Electrical](https://www.mackie-electrical.co.uk/zero-vat-battery-storage-guide)
- [S60] [Battery Storage Without Solar UK 2026: Cost & Payback Guide | HeatPumpsAndSolar](https://heatpumpsandsolar.co.uk/insights/battery-storage-without-solar-uk)
- [S61] [Solar Panels Scotland | Free Quotes from Accredited Installers | Scottish Energy Efficiency](https://scottishenergyefficiency.co.uk/battery-storage-scotland)

[S1]: https://octopus.energy/agile/
[S2]: https://www.energyplus.co.uk/costsavingadvice/smart-meter-tariffs-ranked
[S3]: https://www.energyplus.co.uk/suppliers/octopus-energy
[S4]: https://energysavingtrust.org.uk/time-use-tariffs-all-you-need-know/
[S5]: https://octopus.energy/smart/tracker/
[S6]: https://octopus.energy/blog/april-2026-price-cap-updates/
[S7]: https://switchtooctopus.co.uk/docs/tariffs/understanding-tariffs/
[S8]: https://www.moneysavingexpert.com/utilities/are-there-any-cheap-fixed-energy-deals-currently-worth-it/
[S9]: https://www.energyplus.co.uk/solar/should-i-add-a-battery-to-my-solar-panels-in-2026
[S10]: https://www.ukem.co.uk/solar-battery-storage/
[S11]: https://octopus.energy/
[S12]: https://amprenewables.co.uk/blog/best-time-of-use-tariff-uk-2026
[S13]: https://www.energyplus.co.uk/costsavingadvice/energy-suppliers-offering-free-electricity-hours-tariff-uk
[S14]: https://www.ovoenergy.com/
[S15]: https://cucumbereco.co.uk/blog/heat-pump-tariffs-2026
[S16]: https://www.sunsave.energy/solar-panels-advice/exporting-to-the-grid/best-seg-rates
[S17]: https://www.moneysavingexpert.com/news/2026/05/martin-lewis-energy-price-cap-rise-july/
[S18]: https://www.energyplus.co.uk/solar/home-battery-storage-without-solar-panels-uk
[S19]: https://switchtogether.co.uk/resource-hub/blog/solar-export-tariffs
[S20]: https://octopus.energy/go/
[S21]: https://octopus.energy/policies/smart-tariffs-terms-and-condition/
[S22]: https://octopus.energy/smart/flux/
[S23]: https://octopus.energy/ev-charger-faq/
[S24]: https://octopus.energy/smart/intelligent-octopus-flux/
[S25]: https://octopus.energy/intelligent-flux-faqs/
[S26]: https://www.gov.uk/guidance/vat-on-energy-saving-materials-and-heating-equipment-notice-7086
[S27]: https://www.gov.uk/guidance/the-vat-domestic-reverse-charge-procedure-notice-735
[S28]: https://www.gov.uk/government/publications/warm-homes-plan/warm-homes-plan-html
[S29]: https://www.sunsave.energy/solar-panels-advice/exporting-to-the-grid/best-ev-charging-tariffs
[S30]: https://www.loveelectric.cars/blog/best-ev-tariff-uk
[S31]: https://www.edfenergy.com/solar/solar-tariffs
[S32]: https://octopus.energy/blog/battery-only-installation/
[S33]: https://www.gridguru.co.uk/post/grid-guru-july-2026-newsletter
[S34]: https://www.eonnext.com/energy/guides/time-of-use-tariffs
[S35]: https://solargridcheck.co.uk/best-home-batteries-uk
[S36]: https://www.edfenergy.com/media-centre/edf-launches-britains-first-solar-tariff-no-standing-charge
[S37]: https://www.edfenergy.com/gas-and-electricity/change-energy-tariff
[S38]: https://www.edfenergy.com/energy-efficiency/smart-export-tariff
[S39]: https://www.edfenergy.com/energywise/choosing-best-energy-tariff
[S40]: https://www.edfenergy.com/energywise/energy-price-cap-july-2026
[S41]: https://octopus.energy/blog/should-I-fix-before-july-2026-price-cap/
[S42]: https://www.edfenergy.com/media-centre/edf-launches-first-its-kind-three-rate-tariff-free-electricity-hours
[S43]: https://www.edfenergy.com/electric-cars/tariffs-guide
[S44]: https://www.gov.uk/government/consultations/improving-the-energy-efficiency-of-socially-rented-homes-in-england/outcome/improving-the-energy-efficiency-of-socially-rented-homes-in-england-government-response
[S45]: https://www.edfenergy.com/electric-cars/ev-tariffs
[S46]: https://www.edfenergy.com/media-centre/edf-launches-cheapest-overnight-charging-ev-tariffs
[S47]: https://www.ovoenergy.com/pricecap
[S48]: https://www.edfenergy.com/energy-efficiency/sunday-saver-challenge
[S49]: https://www.edfenergy.com/electric-cars/ev-tariffs/goelectric
[S50]: https://www.ofgem.gov.uk/environmental-and-social-schemes/boiler-upgrade-scheme-bus/boiler-upgrade-scheme-bus-property-owners
[S51]: https://www.bakewelltowncouncil.gov.uk/uploads/bakewell-town-guide-2025.pdf?v=1750841577
[S52]: https://assets.publishing.service.gov.uk/media/60bdd2d2e90e0743ae8c284e/uk-integrated-national-energy-climate-plan-necp-31-january-2020.pdf
[S53]: https://www.gov.uk/government/publications/clean-flexibility-roadmap/clean-flexibility-roadmap-july-2026-update-accessible-webpage
[S54]: https://www.gov.uk/government/publications/clean-flexibility-roadmap/clean-flexibility-roadmap
[S55]: https://www.ofgem.gov.uk/environmental-and-social-schemes/feed-tariffs-fit
[S56]: https://www.gov.uk/government/consultations/smart-secure-electricity-systems-sses-programme-first-phase-energy-smart-appliances-regulations/smart-secure-electricity-systems-sses-programme-first-phase-energy-smart-appliances-regulations-consultation-document-accessible-webpage
[S57]: https://assets.publishing.service.gov.uk/media/6a2041a058aae59498cb29e5/assessing-the-case-for-community-batteries-call-for-evidence.pdf
[S58]: https://www.gov.uk/government/consultations/smart-secure-electricity-systems-programme-sses-enduring-governance/smart-secure-electricity-systems-sses-programme-enduring-governance-accessible-webpage
[S59]: https://www.mackie-electrical.co.uk/zero-vat-battery-storage-guide
[S60]: https://heatpumpsandsolar.co.uk/insights/battery-storage-without-solar-uk
[S61]: https://scottishenergyefficiency.co.uk/battery-storage-scotland

### Analyzed URLs

1. [Agile Octopus | Half-hourly UK Wholesale Price Tracker](https://octopus.energy/agile/)
2. [Smart Meter Tariffs Ranked July 2026: Agile, Go, Cosy & Flux](https://www.energyplus.co.uk/costsavingadvice/smart-meter-tariffs-ranked)
3. [Octopus Energy price changes - July 2026 - Uswitch](https://www.uswitch.com/gas-electricity/guides/octopus-price-changes/)
4. [Octopus Energy Tariffs & Rates UK (Updated July 2026) - EnergyPlus](https://www.energyplus.co.uk/suppliers/octopus-energy)
5. [Smart time of use tariffs: all you need to know - Energy Saving Trust](https://energysavingtrust.org.uk/time-use-tariffs-all-you-need-know/)
6. [Daily UK Wholesale Price Tracker | Octopus Energy](https://octopus.energy/smart/tracker/)
7. [How do I get the energy price cut from April 2026?](https://octopus.energy/blog/april-2026-price-cap-updates/)
8. [Understanding Octopus tariffs](https://switchtooctopus.co.uk/docs/tariffs/understanding-tariffs/)
9. [Should I fix my energy or stay on the Price Cap?](https://www.moneysavingexpert.com/utilities/are-there-any-cheap-fixed-energy-deals-currently-worth-it/)
10. [Octopus Agile rates today UK: guide to half-hour pricing - EnergyPlus](https://www.energyplus.co.uk/news/octopus-agile-tariff-rates-today-uk)
11. [Should I Add a Solar Battery in 2026? UK Guide (July ... - EnergyPlus](https://www.energyplus.co.uk/solar/should-i-add-a-battery-to-my-solar-panels-in-2026)
12. [Solar Battery Storage System for UK Homes | UKEM Group](https://www.ukem.co.uk/solar-battery-storage/)
13. [Octopus Energy: The UK's most awarded energy supplier](https://octopus.energy/)
14. [Best Time-of-Use Tariff UK 2026: Octopus Go vs Agile](https://amprenewables.co.uk/blog/best-time-of-use-tariff-uk-2026)
15. [UK Energy Suppliers Offering Free Electricity Hours Tariffs (July 2026)](https://www.energyplus.co.uk/costsavingadvice/energy-suppliers-offering-free-electricity-hours-tariff-uk)
16. [OVO Energy: Energy Supplier, Switch Gas & Electricity Provider](https://www.ovoenergy.com/)
17. [Best Heat Pump Tariffs in 2026: How to Cut Your ... - Cucumber Eco](https://cucumbereco.co.uk/blog/heat-pump-tariffs-2026)
18. [What are the best SEG rates? | All 38 tariffs ranked [2026] - Sunsave](https://www.sunsave.energy/solar-panels-advice/exporting-to-the-grid/best-seg-rates)
19. [Martin Lewis: For most people, July's 13% Energy Price Cap rise is ...](https://www.moneysavingexpert.com/news/2026/05/martin-lewis-energy-price-cap-rise-july/)
20. [Home Battery Storage Without Solar Panels UK — 2026 Costs](https://www.energyplus.co.uk/solar/home-battery-storage-without-solar-panels-uk)
21. [Solar Export Tariffs in 2026: How to Earn from Your Solar Panels](https://switchtogether.co.uk/resource-hub/blog/solar-export-tariffs)
22. [Octopus Go | Off Peak EV Charging for Any Car & Charger](https://octopus.energy/go/)
23. [Smart Tariffs - Terms and Conditions - Octopus Energy](https://octopus.energy/policies/smart-tariffs-terms-and-condition/)
24. [Octopus Flux | Energy Tariff Designed for Solar & Batteries](https://octopus.energy/smart/flux/)
25. [EV charger installation with Octopus Energy: FAQs](https://octopus.energy/ev-charger-faq/)
26. [Intelligent Octopus Flux | Automated export to support the grid](https://octopus.energy/smart/intelligent-octopus-flux/)
27. [Intelligent Octopus Flux: Frequently Asked Questions](https://octopus.energy/intelligent-flux-faqs/)
28. [Energy-saving materials and heating equipment (VAT Notice 708/6)](https://www.gov.uk/guidance/vat-on-energy-saving-materials-and-heating-equipment-notice-7086)
29. [Domestic reverse charge procedure (VAT Notice 735) - GOV.UK](https://www.gov.uk/guidance/the-vat-domestic-reverse-charge-procedure-notice-735)
30. [Warm Homes Plan (HTML) - GOV.UK](https://www.gov.uk/government/publications/warm-homes-plan/warm-homes-plan-html)
31. [Best energy tariff for home battery storage (UK guide) - EnergyPlus](https://www.energyplus.co.uk/guidesfaqs/best-energy-tariff-for-home-battery-storage-uk)
32. [The 11 best EV tariffs [UK, 2026] - Sunsave](https://www.sunsave.energy/solar-panels-advice/exporting-to-the-grid/best-ev-charging-tariffs)
33. [Compare UK Solar Tariffs - July 2026 - MoneySuperMarket](https://www.moneysupermarket.com/gas-and-electricity/solar-tariffs/)
34. [Best EV Tariff UK 2026: A Driver's Guide to Cheaper Home Charging](https://www.loveelectric.cars/blog/best-ev-tariff-uk)
35. [Solar Tariffs - Compare and Save on Energy Bills | EDF](https://www.edfenergy.com/solar/solar-tariffs)
36. [Introducing battery-only installation with Octopus](https://octopus.energy/blog/battery-only-installation/)
37. [Grid Guru July 2026 Newsletter: Key Updates in UK Solar & Energy ...](https://www.gridguru.co.uk/post/grid-guru-july-2026-newsletter)
38. [Understanding time of use tariffs and energy bills. - E.ON Next](https://www.eonnext.com/energy/guides/time-of-use-tariffs)
39. [Best Home & Solar Batteries UK (2026)](https://solargridcheck.co.uk/best-home-batteries-uk)
40. [EDF launches Britain's first solar tariff with no standing charge](https://www.edfenergy.com/media-centre/edf-launches-britains-first-solar-tariff-no-standing-charge)
41. [Change or Renew Your Energy Tariff | New & Existing Customers](https://www.edfenergy.com/gas-and-electricity/change-energy-tariff)
42. [Smart Export Guarantee & Tariff | Solar Energy | EDF](https://www.edfenergy.com/energy-efficiency/smart-export-tariff)
43. [Choosing the Best Energy Tariffs - How to pick the right one for you](https://www.edfenergy.com/energywise/choosing-best-energy-tariff)
44. [July 2026 Energy Price Cap to rise by 13% | EDF](https://www.edfenergy.com/energywise/energy-price-cap-july-2026)
45. [Should I fix my energy tariff before the July price cap?](https://octopus.energy/blog/should-I-fix-before-july-2026-price-cap/)
46. [EDF launches first-of-its-kind three-rate tariff with free electricity hours](https://www.edfenergy.com/media-centre/edf-launches-first-its-kind-three-rate-tariff-free-electricity-hours)
47. [A guide to electric car tariffs - EDF Energy](https://www.edfenergy.com/electric-cars/tariffs-guide)
48. [Improving the energy efficiency of socially rented homes in England](https://www.gov.uk/government/consultations/improving-the-energy-efficiency-of-socially-rented-homes-in-england/outcome/improving-the-energy-efficiency-of-socially-rented-homes-in-england-government-response)
49. [Electric vehicle tariffs for cheaper home charging - EDF Energy](https://www.edfenergy.com/electric-cars/ev-tariffs)
50. [EDF launches cheapest overnight charging on EV tariffs](https://www.edfenergy.com/media-centre/edf-launches-cheapest-overnight-charging-ev-tariffs)
51. [Energy Price Cap: Everything You Need To Know](https://www.ovoenergy.com/pricecap)
52. [Earn free electricity on Sundays with our Sunday Saver Challenge](https://www.edfenergy.com/energy-efficiency/sunday-saver-challenge)
53. [Overnight EV Tariff For Your Car And Home | GoElectric - EDF Energy](https://www.edfenergy.com/electric-cars/ev-tariffs/goelectric)
54. [Boiler Upgrade Scheme (BUS) - Property owners - Ofgem](https://www.ofgem.gov.uk/environmental-and-social-schemes/boiler-upgrade-scheme-bus/boiler-upgrade-scheme-bus-property-owners)
55. [bakewell-town-guide-2025.pdf](https://www.bakewelltowncouncil.gov.uk/uploads/bakewell-town-guide-2025.pdf?v=1750841577)
56. [The UK's Integrated National Energy and Climate Plan - GOV.UK](https://assets.publishing.service.gov.uk/media/60bdd2d2e90e0743ae8c284e/uk-integrated-national-energy-climate-plan-necp-31-january-2020.pdf)
57. [The best solar batteries of 2026 | Researched and reviewed](https://www.theecoexperts.co.uk/solar-panels/the-best-storage-batteries)
58. [Clean flexibility roadmap: July 2026 update (accessible webpage)](https://www.gov.uk/government/publications/clean-flexibility-roadmap/clean-flexibility-roadmap-july-2026-update-accessible-webpage)
59. [Clean flexibility roadmap (accessible webpage) - GOV.UK](https://www.gov.uk/government/publications/clean-flexibility-roadmap/clean-flexibility-roadmap)
60. [Feed-in Tariffs (FIT) - Ofgem](https://www.ofgem.gov.uk/environmental-and-social-schemes/feed-tariffs-fit)
61. [Get a smart meter | Ofgem](https://www.ofgem.gov.uk/getting-smart-meter)
62. [Smart Secure Electricity Systems (SSES) Programme: first phase ...](https://www.gov.uk/government/consultations/smart-secure-electricity-systems-sses-programme-first-phase-energy-smart-appliances-regulations/smart-secure-electricity-systems-sses-programme-first-phase-energy-smart-appliances-regulations-consultation-document-accessible-webpage)
63. [Assessing the case for community batteries: call for evidence](https://assets.publishing.service.gov.uk/media/6a2041a058aae59498cb29e5/assessing-the-case-for-community-batteries-call-for-evidence.pdf)
64. [Electric Vehicle Smart Charging Action Plan - GOV.UK](https://assets.publishing.service.gov.uk/media/655dfabf046ed400148b9e0a/electric-vehicle-ev-smart-charging-action-plan.pdf)
65. [Smart Secure Electricity Systems (SSES) Programme - GOV.UK](https://www.gov.uk/government/consultations/smart-secure-electricity-systems-programme-sses-enduring-governance/smart-secure-electricity-systems-sses-programme-enduring-governance-accessible-webpage)
66. [Guidance for Generators - Ofgem](https://www.ofgem.gov.uk/sites/default/files/2024-09/Guidance_for_FIT_Generators_V18.pdf)
67. [Zero VAT on Home Battery Storage: The HMRC Rules, Real ...](https://www.mackie-electrical.co.uk/zero-vat-battery-storage-guide)
68. [Battery Storage Without Solar UK 2026: Cost & Payback Guide](https://heatpumpsandsolar.co.uk/insights/battery-storage-without-solar-uk)
69. [Battery Storage Scotland: Home Battery Costs & Savings (2026)](https://scottishenergyefficiency.co.uk/battery-storage-scotland)

<details>
<summary><strong>Raw collected findings (61 sources)</strong></summary>

**1. [S1] [Agile Octopus | Half-hourly UK Wholesale Price Tracker | Octopus Energy](https://octopus.energy/agile/)**

The page suggests that a battery could be useful without solar only if it can reliably charge during cheap periods and discharge during expensive periods, but the household's unusually high daytime office demand weakens the case because much of its consumption occurs outside the tariff's typical 4–7pm peak and may not be available for battery discharge. Agile's extreme volatility, possible prices up to 100 p/kWh, negative-price events, and the supplier's warning that Agile may currently cost more than a standard tariff make tariff selection and actual half-hourly usage data essential. The page supports considering a smart tariff without a battery, but it cannot establish annual costs, payback, 10-year net outcomes, or whether a 5 kWh or 10 kWh battery will pay for itself. Those calculations require current tariff quotes and standing charges, the home's half-hourly consumption profile, battery usable capacity and charge/discharge limits, round-trip efficiency, installed quotations, warranty and degradation terms, replacement assumptions, and confirmation of applicable UK tax and market rules.

**2. [S2] [Smart Meter Tariffs Ranked July 2026: Agile, Go, Cosy & Flux](https://www.energyplus.co.uk/costsavingadvice/smart-meter-tariffs-ranked)**

For the stated home, the webpage supports a cautious conclusion that switching to time-of-use without a battery is worthwhile only if a meaningful share of the 9,000 kWh annual demand can be moved into cheap periods. The unusually high daytime office demand may avoid some evening peak exposure, but the page does not define the tariff’s actual peak hours, so the household’s half-hourly smart-meter data is essential. A battery could potentially charge overnight and discharge during expensive periods, but this source contains no evidence needed to model whether a 5 kWh or 10 kWh system would pay for itself. In particular, it supplies no installed-price quotations, battery usable-capacity or power limits, round-trip efficiency, degradation, warranty, maintenance, replacement or export assumptions. Consequently, no defensible annual battery costs, simple paybacks, sensitivity analysis or 10-year net outcomes can be calculated from this webpage alone. The strongest usable evidence is the warning that time-of-use tariffs can increase bills when consumption remains in peak periods, contrasted with the illustrative saving available when large loads are reliably shifted overnight.

**3. [S3] [Octopus Energy Tariffs & Rates UK (Updated July 2026)](https://www.energyplus.co.uk/suppliers/octopus-energy)**

The webpage supports switching away from a 24.7p/kWh flat tariff only if a postcode-specific competitive tariff is genuinely cheaper after standing charges; its cited 22.85p/kWh starting price suggests a possible saving of roughly £167/year on 9,000 kWh, while its 27.78p/kWh regional-average fixed rate would be worse. A time-of-use tariff without a battery may provide meaningful savings only where enough consumption can be moved into documented cheap periods. This household’s high daytime office demand and limited flexibility weaken that case, and the page explicitly warns that heavy 4–7pm usage can make smart tariffs more expensive. A 5 kWh or 10 kWh battery cannot be judged financially from this webpage: no suitable full tariff rates, battery prices, technical specifications or degradation data are supplied. The likely best financial option on the available evidence is a postcode-checked competitive flat tariff, followed by a time-of-use tariff only after analysing half-hourly smart-meter data. A battery could become worthwhile if the homeowner obtains a large and durable peak/off-peak spread, can cycle the battery regularly without forcing daytime loads onto expensive periods, and receives a low installed quote; the 10 kWh model would require substantially more shifted demand and therefore is not automatically better. The strongest argument against buying a battery is that the home’s electricity is unusually concentrated in daytime office hours, while overnight charging may merely replace electricity that would otherwise be consumed directly and battery losses reduce the arbitrage margin. Before deciding, obtain 12 months of half-hourly consumption data, tariff information labels showing every time band and standing charge, a postcode-specific flat-tariff quote, and at least three itemised battery quotations stating usable capacity, continuous and peak charge/discharge power, round-trip efficiency, degradation or throughput warranty, inverter warranty, replacement assumptions, maintenance, VAT treatment, export capability and backup-installation cost.

**4. [S4] [Smart time of use tariffs: all you need to know - Energy Saving Trust](https://energysavingtrust.org.uk/time-use-tariffs-all-you-need-know/)**

The source supports the following provisional conclusion. Option 2—a time-of-use tariff without a battery—could provide meaningful savings only for the portion of the household’s 6,000–11,000 kWh annual demand that can be moved into cheap periods. Because the home office uses 10–20 kWh per day mainly during daytime working hours and only a small amount of consumption can be shifted manually, the webpage’s warning that a time-of-use tariff is unlikely to work when demand cannot be moved is particularly important. A competitive flat tariff may therefore be safer than a tariff with expensive peak pricing unless the actual off-peak discount exceeds the cost of unavoidable peak consumption.

Options 3 and 4 could exploit the tariff mechanically: the battery could charge overnight and discharge during expensive periods. However, the source gives no evidence that a 5 kWh or 10 kWh battery would cycle sufficiently often for savings to recover installation costs. The daytime office load may allow substantial direct daytime consumption rather than battery discharge, while a battery cannot eliminate peak-period electricity used beyond its usable capacity or maximum discharge rate. The correct calculation must therefore use half-hourly smart-meter data, actual usable capacity, charge and discharge limits, round-trip efficiency, charging schedule, and the proportion of energy that would otherwise be bought at peak prices—not assume a full daily cycle.

No defensible annual electricity costs, annual savings, simple payback, or 10-year net outcome can be calculated from this webpage alone: it contains no current England tariff prices, flat-rate comparator, battery installation quotations, warranty or replacement costs, degradation rates, maintenance costs, VAT or incentive treatment, export rules, or supplier eligibility details. The strongest source-based recommendation is to obtain those inputs before purchasing. Financially, the battery is unlikely to be worthwhile unless the achievable peak/off-peak spread is large and durable, the battery is used frequently without excessive degradation, and the installed price is low. The no-battery time-of-use option is likely to be the best low-risk first step only if actual half-hourly data shows that enough discretionary usage can be shifted. The strongest argument against buying either battery is that the household’s unusually high daytime demand and limited manual flexibility may leave too little expensive-period consumption for storage arbitrage to repay the capital cost. The homeowner should obtain at least 12 months of half-hourly smart-meter data; bills and standing charges; prices, time windows, seasonal rules and eligibility for several static and dynamic tariffs; and itemised quotations for 5 kWh and 10 kWh systems showing usable capacity, continuous and peak charge/discharge power, round-trip efficiency, warranty throughput and duration, degradation guarantee, inverter replacement risk, installation work, VAT, maintenance, outage-backup capability, and battery replacement assumptions.

**5. [S5] [Octopus Tracker | Daily UK Wholesale Price Tracker | Octopus Energy](https://octopus.energy/smart/tracker/)**

The page supports considering a smart-meter-dependent variable tariff as an alternative to a flat tariff, especially because prices can fall immediately when wholesale prices fall. It also warns that Tracker prices can rise substantially in colder months and that its 100p/kWh electricity ceiling is much higher than the normal price cap, making tariff risk important. For the stated household, the unusually high daytime office demand may reduce the amount of cheap overnight electricity that can be stored and later discharged during expensive periods, so this webpage alone cannot establish that a battery is worthwhile. No indicative annual costs, battery savings, payback, or 10-year net outcome can be calculated reliably from the supplied content: those require actual Tracker price history or a quoted tariff, half-hourly household consumption data, battery usable capacity and charge/discharge limits, round-trip efficiency, installed quotations, warranty and replacement terms, and assumptions about degradation and tariff changes. The page does establish that a compatible smart meter and half-hourly data are prerequisites for this tariff, and that the strongest non-battery comparison should include both a protected flat tariff and Tracker's variable-price risk.

**6. [S6] [How do I get the energy price cut from April 2026? | Octopus Energy](https://octopus.energy/blog/april-2026-price-cap-updates/)**

The source supports using April 2026 prices rather than the household’s current £0.247/kWh rate without adjustment: policy-cost reductions are automatically reflected in Octopus tariffs, including smart tariffs, and the benefit is intended to continue until 2029. However, the page gives only typical dual-fuel bill changes, not the electricity unit rates or peak/off-peak spread needed to compare a flat tariff with a time-of-use tariff. It therefore provides context for tariff uncertainty and policy changes but no evidence that a 5 kWh or 10 kWh battery would pay for itself. A proper comparison requires current tariff quotations, half-hourly consumption data, battery installed prices and technical specifications, and assumptions for losses, degradation, warranty and replacement.

**7. [S7] [Understanding Octopus tariffs | Switch to Octopus](https://switchtooctopus.co.uk/docs/tariffs/understanding-tariffs/)**

For this England household, the page supports the general conclusion that a smart meter and a tariff with genuinely cheap overnight or half-hourly periods are prerequisites for evaluating battery arbitrage, while regional standing charges and tariff eligibility must be checked using a postcode-specific quote. The household’s high daytime usage may reduce battery value because much consumption occurs when electricity is already being used directly rather than shifted from overnight storage. The page does not establish that a 5 kWh or 10 kWh battery will pay for itself: it contains no installed-price evidence, usable capacity, charge/discharge limits, round-trip efficiency, degradation, warranty, replacement, maintenance, or consumption-profile data. It also does not identify a currently available general-purpose battery-only tariff; Flux is presented as a solar-and-battery route, and Intelligent Flux is temporarily unavailable. Consequently, the source can inform tariff screening but cannot support the requested annual-cost, payback, sensitivity, or 10-year financial calculations. Before deciding, the homeowner should obtain postcode-specific tariff terms, confirm that battery charging is permitted and technically compatible, download at least several months of half-hourly smart-meter data, and obtain comparable installed quotations stating usable capacity, power rating, efficiency, warranty, degradation guarantee, maintenance, replacement assumptions, and backup functionality.

**8. [S8] [Is it time to fix your energy or stay on the Price Cap? Martin Lewis](https://www.moneysavingexpert.com/utilities/are-there-any-cheap-fixed-energy-deals-currently-worth-it/)**

Using the household’s stated electricity unit rate, the flat-rate energy-cost component is approximately £2,223 per year (9,000 kWh × £0.247), before any standing charge. The webpage suggests that switching to a suitably discounted fixed tariff could materially reduce costs, but its examples are not electricity-only prices for this household and cannot establish the saving for a 9,000 kWh user. It supplies no time-of-use rates or battery evidence, so annual costs, payback, degradation-adjusted 10-year outcomes, and a 5 kWh versus 10 kWh recommendation cannot be calculated from this webpage. The homeowner would need region-specific electricity-only tariff quotes—including peak, off-peak and standing-charge rates—and battery quotations specifying installed price, usable capacity, power limits, round-trip efficiency, warranty, degradation guarantee, replacement terms and export/backup functionality. The supplied page supports investigating a cheaper tariff first; it does not support concluding that either battery size will pay for itself.

**9. [S9] [Should I Add a Solar Battery in 2026? UK Guide (July 2026)](https://www.energyplus.co.uk/solar/should-i-add-a-battery-to-my-solar-panels-in-2026)**

The webpage supports a cautious conclusion: a battery without solar could only earn money through tariff arbitrage, not solar self-consumption. For each 1 kWh delivered during an expensive period, the approximate gross saving would be the peak import price minus the off-peak charging cost divided by round-trip efficiency; the result must then be reduced for unused capacity, direct overnight consumption, battery degradation, cycling limits and replacement risk. The household’s high daytime office demand weakens the case because it leaves less flexible peak-period consumption for discharge, and a 5 kWh battery may be more likely to cycle usefully than a 10 kWh battery unless measured peak demand is high. Changing to a suitable time-of-use tariff could provide savings without a battery, but the webpage supplies no tariff prices, so it cannot establish whether those savings are meaningful. It also cannot calculate the requested 6,000-, 9,000- and 11,000-kWh scenarios, installed-price payback or 10-year net outcomes. Before deciding, obtain half-hourly smart-meter data, especially overnight charging potential and 4pm–11pm consumption; actual tariff terms and peak/off-peak rates; battery usable capacity, charge/discharge power and efficiency; installed quotations including inverter, protection, backup and DNO work; warranty cycle and end-capacity guarantees; and a model showing monthly cycles, degradation and replacement assumptions. The strongest argument against buying is that the achievable tariff spread and peak-period discharge may be too small or too infrequent to repay the capital cost over ten years, whereas switching tariff has little or no upfront cost.

**10. [S10] [Solar Battery Storage System for UK Homes | UKEM Group](https://www.ukem.co.uk/solar-battery-storage/)**

{
  "rational": "The webpage is relevant because it describes batteries that can be added to an existing system or charged from cheap tariffs, and it provides technical information for approximately 5 kWh and 10 kWh battery choices. However, most of its savings and payback claims concern solar-plus-battery systems, not a battery-only installation. It does not provide the tariff prices, standalone installed prices, demand-profile assumptions, charge/discharge limits, degradation modelling, replac

**11. [S11] [Octopus Energy: The UK's most awarded energy supplier](https://octopus.energy/)**

The strongest relevant evidence is that a smart meter enables time-of-use options: six nightly off-peak hours on Intelligent Octopus Go and potentially cheaper 4pm–7pm periods on Agile Octopus. For this household, a tariff-only switch could provide savings only for electricity that can actually be moved into cheaper periods; the unusually high daytime office demand limits that opportunity. A battery could charge overnight and discharge during expensive periods, but this webpage does not supply enough evidence to calculate annual costs, payback, or a 10-year outcome for 5 kWh versus 10 kWh. The page’s battery claims are tied to solar and should not be used to justify a battery-only purchase. Obtain the supplier’s current tariff unit rates and standing charges, the home’s half-hourly smart-meter consumption data, and several comparable installed battery quotations specifying usable capacity, power rating, efficiency, warranty, degradation guarantee, replacement terms, controls, and backup capability before making a decision.

**12. [S12] [Best Time-of-Use Tariff UK 2026: Octopus Go vs Agile](https://amprenewables.co.uk/blog/best-time-of-use-tariff-uk-2026)**

{
  "rational": "The webpage directly compares UK time-of-use tariffs for households with a battery but no solar, identifying Octopus Go as its preferred option and giving indicative off-peak and peak prices. It also discusses battery-only savings, battery size, tariff suitability, smart-meter requirements, and the limitations of Agile. However, it does not provide independently verified installed battery prices, manufacturer degradation data, warranty terms, charge/discharge limits, maintenance

**13. [S13] [UK Energy Suppliers Offering Free Electricity Hours Tariffs (July 2026)](https://www.energyplus.co.uk/costsavingadvice/energy-suppliers-offering-free-electricity-hours-tariff-uk)**

The webpage supports a cautious conclusion that switching from a flat tariff to a TOU tariff could reduce costs without a battery only if the household can shift substantial consumption into genuinely cheap or free periods. That is a concern here because much of the 10–20 kWh/day office demand occurs during daytime working hours and only limited manual shifting is possible. A battery could shift some overnight electricity to peak periods, but the page provides no data for quantifying the benefit after round-trip losses, usable capacity, power limits, degradation, or the fact that overnight electricity may be consumed directly rather than stored. Therefore no defensible annual cost, simple payback, or 10-year net outcome for 5 kWh versus 10 kWh can be calculated from this source alone. The page’s strongest relevant recommendation is to obtain actual half-hourly consumption data, compare the complete tariff—including peak, off-peak and standing-charge rates—and verify the supplier’s precise rules before purchase. A battery is more likely to be worthwhile where the peak/off-peak spread is large, the battery can cycle regularly without excessive degradation, installation cost is low, and enough daytime or evening demand exists to use the stored energy; the case weakens where the household consumes electricity directly overnight, has high daytime demand that cannot be shifted, or faces high peak rates and battery losses.

**14. [S14] [Energy Supplier, Switch Gas & Electricity Provider | OVO Energy](https://www.ovoenergy.com/)**

This webpage does not support a financial comparison of the four requested options. It confirms only that OVO markets solar-plus-battery packages and quotes a historical standard-variable electricity rate in an unrelated EV example; it supplies no evidence that a battery without solar would be worthwhile. The homeowner would need current tariff terms and half-hourly consumption data, plus comparable installed quotations for 5 kWh and 10 kWh batteries, before calculating annual savings, payback, or a 10-year outcome. In particular, the quoted solar-and-battery savings claim should not be used for this household because its assumptions differ substantially from the stated 9,000 kWh usage and high daytime demand.

**15. [S15] [Best Heat Pump Tariffs in 2026: How to Cut Your Heat Pump Running Costs With Time of Use Electricity | Cucumber Eco](https://cucumbereco.co.uk/blog/heat-pump-tariffs-2026)**

This webpage supports only the general conclusion that a smart meter and time-of-use pricing can create savings by moving electricity from expensive periods to cheap periods, and that battery optimisation may involve retaining a reserve and accounting for only part of the nominal battery capacity. It does not establish a suitable tariff for a gas-heated English home without solar, nor does it contain enough evidence to calculate annual costs, battery payback, sensitivity analysis, or a 10-year financial result for 5 kWh and 10 kWh batteries. The quoted Cosy Octopus rates are from 1 April–30 June 2026, not specifically 14 July 2026, and the tariff requires a heat pump, electric boiler, or electric radiator according to the page. Further research should use current supplier eligibility and tariff pages, Ofgem data, manufacturer technical documentation, and independent battery-cost and degradation evidence before recommending either battery size.

**16. [S16] [What are the best SEG rates? | All 38 tariffs ranked [2026]](https://www.sunsave.energy/solar-panels-advice/exporting-to-the-grid/best-seg-rates)**

The strongest directly relevant evidence is that July 2026 time-of-use tariffs can offer substantial spreads: approximately 10.361p/kWh overnight versus 30.22p/kWh during EDF’s 4pm–7pm peak, or roughly 7.43p/kWh overnight versus up to 46.96p/kWh on 100green’s peak period. A battery-only system could therefore save money by charging from the grid overnight and discharging during expensive periods, but the webpage does not establish that either a 5 kWh or 10 kWh battery will pay for itself. Its headline savings are solar-specific and include export income, while the stated household has unusually high daytime office demand, which may reduce the amount of energy available for profitable peak discharge. A meaningful assessment requires interval smart-meter data showing consumption during each tariff period, the tariff’s standing charges and eligibility rules, a battery quotation with usable capacity, maximum power, efficiency, degradation and warranty details, and a comparison of annual savings after losses and installation costs. On this source alone, switching to a suitable time-of-use tariff without buying a battery is the lower-risk option; a battery is financially attractive only if a large proportion of demand can be shifted from expensive periods, the spread remains wide, the battery is competitively priced, and cycling is sufficient without requiring unsupported assumptions of one full cycle every day.

**17. [S17] [Martin Lewis: For most people, July’s 13% Energy Price Cap rise is voluntary – it can (and should) be avoided](https://www.moneysavingexpert.com/news/2026/05/martin-lewis-energy-price-cap-rise-july/)**

The page supports switching away from an expensive standard variable or capped tariff: its published average electricity rate rises from 24.67p/kWh to 26.11p/kWh on 1 July 2026, including VAT, while Martin Lewis says competitive fixes may be below the cap and recommends a whole-of-market comparison. For the stated 9,000 kWh usage, the published unit-rate difference is approximately £127 per year before standing charges, so tariff switching could provide meaningful savings if a genuinely cheaper tariff is available. It does not establish that a battery is worthwhile, because it contains no time-of-use tariff prices or battery-cost and performance evidence. In particular, the household’s high daytime consumption may leave relatively little electricity available for profitable peak-time discharge, making a battery less attractive than for a household with substantial evening peak demand. A proper comparison requires actual half-hourly smart-meter data, the candidate tariff’s import and export rates and rules, battery usable capacity and power limits, round-trip efficiency, installed quotations, warranty and degradation terms, and a realistic charging schedule rather than assuming one full cycle every day.

**18. [S18] [Home Battery Storage Without Solar Panels UK — 2026 Costs](https://www.energyplus.co.uk/solar/home-battery-storage-without-solar-panels-uk)**

{
  "rational": "The webpage directly addresses grid-charged batteries without solar, including tariff arbitrage, indicative July 2026 electricity prices, installed costs, efficiency, VAT, export limitations, warranties and tariff eligibility. It is useful for constructing an indicative comparison, but it does not provide enough household-specific data to prove that a battery will pay back for this unusually daytime-heavy home. In particular, its headline savings assume one full cycle every day 

**19. [S19] [Solar Export Tariffs in 2026: How to Earn from Your Solar Panels](https://switchtogether.co.uk/resource-hub/blog/solar-export-tariffs)**

This webpage supports only a narrow conclusion: a smart meter enables access to some time-dependent energy arrangements, and tariff rates can vary substantially and change over time. It does not substantiate a financial case for installing a 5 kWh or 10 kWh battery without solar panels. In particular, the cited SEG rates cannot be treated as available battery savings, because SEG is an export-payment scheme for eligible renewable generators. The requested annual costs, payback periods, 10-year outcomes, usage sensitivities, degradation analysis, and comparison with flat or import time-of-use tariffs cannot be calculated reliably from this page. To complete the assessment, obtain current supplier-specific import tariff prices and charging rules, battery quotations including usable capacity and inverter power, warranty and throughput limits, round-trip efficiency, degradation and replacement assumptions, and half-hourly household consumption data—especially the proportion of demand that can actually be served during peak periods.

**20. [S20] [Octopus Go | Off Peak EV Charging for Any Car & Charger | Octopus Energy](https://octopus.energy/go/)**

Using the household figures supplied by the requester, the current flat-rate electricity cost is approximately £2,223 per year (9,000 kWh × £0.247/kWh), before standing charges. The webpage confirms that a smart tariff offers a five-hour overnight window and requires a compatible smart meter, but Octopus Go specifically requires an electric car charged at home; the page does not establish eligibility for charging a standalone 5 kWh or 10 kWh home battery. It also supplies no rates from which to calculate the cost of options 2–4. Consequently, no defensible annual savings, payback, sensitivity analysis, or 10-year net outcome can be derived from this webpage alone. The relevant next step is to obtain current regional tariff rates and written confirmation from the supplier that battery charging is permitted, then model the household's half-hourly demand, usable battery capacity, charge/discharge limits, losses, degradation, installation quotation, warranty, and replacement risk. Given the unusually high daytime office demand, a battery may have limited value because much electricity is consumed when it is generated or purchased, while tariff switching could still save money if the overnight-to-daytime spread is large and the tariff does not impose restrictive EV-only conditions.

**21. [S21] [Smart Tariffs - Terms and Conditions | Octopus Energy](https://octopus.energy/policies/smart-tariffs-terms-and-condition/)**

For the stated household, the page confirms that a smart meter and half-hourly data are relevant to smart tariffs, and that a battery-specific tariff such as Intelligent Octopus Flux requires an Octopus-approved home battery and applies a special 16:00–19:00 rate. Agile exposes the customer to wholesale-linked price volatility, so savings cannot safely be extrapolated from historical prices. Intelligent Octopus Go should not be treated as an available battery-only option because its terms require a primarily home-charged EV or plug-in hybrid. The unusually high daytime office demand may reduce the amount of energy available to charge overnight and discharge during the evening flux period, making actual battery utilisation—and therefore payback—dependent on half-hourly consumption data. A proper financial comparison requires current tariff quotations, standing charges, the battery’s usable capacity and power limits, round-trip efficiency, installed price, warranty/degradation terms, and a model of actual overnight, daytime, and 16:00–19:00 consumption. On this source alone, switching tariff without a battery may be worthwhile only if the household can place meaningful consumption in cheaper periods; the battery case remains unquantified and carries tariff, control, and technology risks explicitly acknowledged by Octopus.

**22. [S22] [Octopus Flux | Energy Tariff Designed for Solar & Batteries | Octopus Energy](https://octopus.energy/smart/flux/)**

The page supports the general principle that a battery can charge during a 02:00–05:00 low-price window and discharge during the 16:00–19:00 high-price window, with manual scheduling and no exit fee or tie-in. But Octopus Flux is explicitly designed for solar-and-battery owners and requires solar, an Export MPAN and an import/export setup; it is therefore not an available tariff for the stated no-solar scenario. Its flexible prices and standing charges may change, and the page gives no numerical rates, so no credible annual-cost, payback or 10-year-net-benefit calculation can be extracted from this source. For the requested comparison, the homeowner would need a battery-compatible import-only time-of-use tariff, current tariff quotations, half-hourly consumption data, and independent battery installation and warranty quotations. The unusually high daytime office demand would also reduce savings because electricity used directly during the day cannot be shifted from the overnight window unless the battery is sufficiently charged and can discharge at the required rate.

**23. [S23] [EV charger installation with Octopus Energy: FAQs | Octopus Energy](https://octopus.energy/ev-charger-faq/)**

This webpage cannot substantively answer whether a 5 kWh or 10 kWh battery is worthwhile without solar panels, nor can it support the requested annual-cost, payback, 10-year-outcome, or sensitivity calculations. The only potentially relevant indication is that Octopus offers a smart-meter-linked tariff with cheap overnight electricity, but the page does not give the tariff rates or explain whether the tariff permits or rewards battery arbitrage. Battery-specific research would require current tariff rates, installed battery quotations, manufacturer technical and warranty documents, and independent evidence on degradation and operating performance.

**24. [S24] [Intelligent Octopus Flux | Automated export to support the grid | Octopus Energy](https://octopus.energy/smart/intelligent-octopus-flux/)**

The source supports only three usable conclusions: the proposed household's smart meter is relevant to smart tariffs; a 16:00–19:00 peak window is the intended discharge period; and Intelligent Octopus Flux is unavailable or unsuitable because it requires solar panels and was temporarily unavailable due to price volatility. It does not support the requested numerical comparison of a flat tariff, a time-of-use tariff, or 5 kWh and 10 kWh batteries. On this evidence alone, no defensible annual cost, simple payback, 10-year net outcome, or sensitivity analysis can be calculated. The strongest source-based caution is that the advertised “up to £300/year” profit is a solar-and-storage marketing claim, not evidence for a grid-charged battery without solar. To complete the assessment, obtain current non-solar tariff rates by half-hour, standing charges, eligibility and export rules; at least 12 months of smart-meter half-hour consumption data; battery quotations including usable capacity, maximum charge/discharge power, installed price, VAT treatment, warranty throughput and end-of-warranty capacity; and supplier confirmation of whether charging from the grid and subsequent discharge to household loads is permitted.

**25. [S25] [Intelligent Octopus Flux: Frequently Asked Questions | Octopus Energy](https://octopus.energy/intelligent-flux-faqs/)**

This webpage provides a strong negative finding for the proposed tariff route: Intelligent Octopus Flux is not presented as a battery-only tariff because eligibility requires both a solar system and a compatible home battery, plus an import/export tariff and half-hourly smart-meter data. It confirms that the tariff uses variable peak/off-peak pricing, with a 16:00–19:00 peak period, supplier-controlled charging and discharging, a stated 20% minimum battery level, and no exit fee. However, the page does not give the actual import prices needed to compare a flat tariff with time-of-use charging, and its £11,115 figure is for a 12-panel solar installation plus a 10 kWh battery rather than a battery-only system. Its 9–13-year payback claim concerns a solar system and cannot be applied to the requested 5 kWh or 10 kWh battery-only options. Consequently, this source alone cannot establish annual costs, savings, payback, or 10-year net outcomes for the 6,000, 9,000, and 11,000 kWh scenarios. The homeowner would need half-hourly consumption data, current eligible battery-only tariff rates, and itemised quotations specifying usable capacity, power limits, efficiency, warranty, degradation, replacement assumptions, and installation cost.

**26. [S26] [Energy-saving materials and heating equipment (VAT Notice 708/6) - GOV.UK](https://www.gov.uk/guidance/vat-on-energy-saving-materials-and-heating-equipment-notice-7086)**

For this England household, the strongest relevant point is a possible VAT advantage: an installed battery may qualify for 0% VAT through 31 March 2027, reverting to 5% thereafter, subject to the installation meeting HMRC's conditions. The supplied webpage does not say that the battery must be paired with solar panels, but it also does not provide enough detail to confirm the treatment of every standalone battery configuration; the installer should confirm the invoice and eligibility. It does not support calculations of annual costs, payback, or 10-year outcomes for the four requested options. Those calculations require current flat and time-of-use tariff rates, interval consumption data, battery quotations, usable capacity, charge/discharge limits, round-trip efficiency, degradation and warranty terms, and an allowance for future VAT and tariff changes. Consequently, no evidence-based conclusion about whether a 5 kWh or 10 kWh battery pays for itself can be drawn from this webpage alone.

**27. [S27] [Domestic reverse charge procedure (VAT Notice 735) - GOV.UK](https://www.gov.uk/guidance/the-vat-domestic-reverse-charge-procedure-notice-735)**

For the stated household, ordinary electricity bought under a domestic supply contract or metered arrangement is outside the domestic VAT reverse charge. This source provides no usable basis for calculating annual electricity costs, battery savings, payback, or a 10-year outcome for 5 kWh versus 10 kWh batteries. Additional primary sources are required, including current domestic tariff prices and terms, Ofgem rules and price-cap information, battery manufacturer specifications and warranties, and verified installed quotations. The webpage therefore contributes only a limited tax clarification: the reverse-charge rules for wholesale electricity do not apply to the household’s normal electricity consumption.

**28. [S28] [Warm Homes Plan (HTML) - GOV.UK](https://www.gov.uk/government/publications/warm-homes-plan/warm-homes-plan-html)**

This source supports the general proposition that batteries and flexible tariffs are part of UK government policy and that future low- or zero-interest finance may improve battery economics. It does not support a numerical comparison of a flat tariff, a time-of-use tariff, or 5 kWh versus 10 kWh batteries. In particular, it provides no current electricity price spreads, standing charges, battery round-trip efficiency, usable capacity, charge or discharge limits, installed prices, warranty or replacement assumptions, degradation rates, export rules, or evidence about whether the household’s daytime-heavy office demand leaves enough peak-period consumption for regular battery cycling. Therefore, the requested annual costs, payback periods, 10-year outcomes, and sensitivity analysis cannot be calculated reliably from this webpage alone. Additional primary evidence should be obtained from Ofgem and suppliers for tariff terms and market rules, government sources for confirmed grants or finance eligibility, and manufacturer quotations and technical documentation for installed cost, usable capacity, power rating, efficiency, warranty, degradation, maintenance, and replacement assumptions.

**29. [S29] [The 11 best EV tariffs [UK, 2026]](https://www.sunsave.energy/solar-panels-advice/exporting-to-the-grid/best-ev-charging-tariffs)**

{
  "rational": "The webpage is relevant mainly because it provides current UK time-of-use tariff rates, off-peak periods, standing charges, eligibility conditions, and an assumed household off-peak usage share. It does not provide the battery prices, technical specifications, degradation data, warranties, replacement costs, or independent performance evidence needed to determine whether a 5 kWh or 10 kWh battery without solar would pay for itself. The calculations below therefore show what can 

**30. [S30] [Best EV Tariff UK 2026: A Driver's Guide to Cheaper Home Charging | loveelectric](https://www.loveelectric.cars/blog/best-ev-tariff-uk)**

This webpage cannot support the requested financial comparison of a flat-rate tariff, a time-of-use tariff, or 5 kWh and 10 kWh home batteries. No indicative costs, savings, payback periods, or 10-year outcomes can be calculated responsibly from the supplied material. Relevant sources would be required, including current supplier tariff terms, Ofgem and government information, manufacturer technical documentation, and itemised installed battery quotations.

**31. [S31] [Solar Tariffs - Compare and Save on Energy Bills | EDF](https://www.edfenergy.com/solar/solar-tariffs)**

Using the user’s flat-rate assumption, the no-battery baseline is approximately £2,223 per year before standing charges or any tariff-specific adjustments (9,000 kWh × £0.247/kWh). The EDF page indicates that a battery-only customer may access a 12-month fixed tariff with a 10p/kWh overnight discount, provided the battery is purchased from EDF and the smart meter provides half-hourly readings. Nevertheless, the page does not provide enough information to calculate annual costs, simple payback, or a 10-year net outcome for either battery size: the relevant off-peak and peak import rates, standing charge, battery installation cost, usable capacity, power limits, round-trip efficiency, degradation, warranty, maintenance, and replacement cost are missing. The strongest page-supported conclusion is therefore that changing to the qualifying tariff may be worthwhile even without solar if the overnight discount and peak/off-peak spread apply to substantial charging and discharge, whereas a battery is unlikely to be demonstrably worthwhile from this source alone. The unusually high daytime office demand weakens the case because much consumption occurs before peak periods and may be cheaper to use directly than to route through a battery. A proper comparison requires half-hourly smart-meter data, the complete tariff schedule and eligibility terms, and at least two installed quotations specifying usable kWh, maximum charge/discharge kW, round-trip efficiency, warranty throughput or cycle limit, degradation guarantee, replacement terms, VAT, and standing charges. The homeowner should model actual eligible charging and discharge rather than assume 365 full cycles. Indicative calculations should use: annual battery saving = eligible discharged kWh × peak/off-peak price spread, reduced for charging losses and degradation, minus any tariff standing-charge difference; simple payback = installed cost divided by first-year net saving; and 10-year net outcome = cumulative energy savings minus installation, maintenance, and any replacement cost. Under the supplied evidence, option 2 (a suitable time-of-use tariff without buying a battery) is the only option that can reasonably be assessed as potentially low-risk; options 3 and 4 require external tariff and quotation data before either can be recommended.

**32. [S32] [Introducing battery-only installation with Octopus | Octopus Energy](https://octopus.energy/blog/battery-only-installation/)**

{
  "rational": "The webpage is directly relevant because it addresses battery-only installation, charging from the grid during off-peak periods, tariff-based savings, available battery sizes, installation prices, and the technical products offered by Octopus. However, it does not provide actual Agile Octopus prices, a representative household demand profile, usable capacity, charge/discharge limits, round-trip efficiency, degradation assumptions, warranty terms, maintenance costs, replacement c

**33. [S33] [Grid Guru July 2026 Newsletter: Key Updates in UK Solar & Energy Market](https://www.gridguru.co.uk/post/grid-guru-july-2026-newsletter)**

{
  "rational": "The webpage is relevant because it reports July 2026 UK electricity prices, smart/off-peak tariff examples, battery-related tariff conditions, VAT treatment, and electricity-network rules. However, it is an installer newsletter rather than an independent battery trial or supplier tariff specification. It does not provide installed battery prices, usable capacities, charge/discharge limits, degradation curves, warranties, replacement costs, maintenance costs, or a general-purpose

**34. [S34] [Understanding time of use tariffs and energy bills.](https://www.eonnext.com/energy/guides/time-of-use-tariffs)**

The page indicates that switching from a flat tariff to a suitable time-of-use tariff can reduce costs when consumption is shifted into cheaper overnight periods, and the stated household already meets the key smart-meter requirement. Nevertheless, the household’s high daytime office demand is important: electricity used directly during daytime peak or shoulder periods may not benefit from the tariff, while a battery could potentially charge overnight and discharge later. The page does not establish whether a 5 kWh or 10 kWh battery would pay for itself, because it supplies none of the required financial or technical inputs. The evidence supports testing the tariff without a battery first and using half-hourly smart-meter data to measure the household’s actual peak, shoulder and off-peak consumption. A battery recommendation would require current tariff quotations, standing charges, export or charging rules, installed battery quotations, usable capacity, maximum charge/discharge power, round-trip efficiency, degradation and warranty terms. No annual electricity costs, savings, payback or 10-year net outcome can be responsibly calculated from this webpage alone.

**35. [S35] [Best Solar Batteries UK 2026: 7 Home Batteries Ranked by £/kWh | SolarGridCheck.co.uk](https://solargridcheck.co.uk/best-home-batteries-uk)**

{
  "rational": "The webpage is directly relevant because it provides 2026 UK installed battery prices, usable capacities, warranties, tariff-arbitrage assumptions, estimated annual savings, payback periods, VAT treatment, and grid-connection requirements. It supports a preliminary comparison, but it does not provide enough evidence to model this particular England household precisely: it gives no charge/discharge limits for the 5 kWh or approximately 10 kWh systems, no measured household demand

**36. [S36] [Britain's first solar tariff with no standing charge launched](https://www.edfenergy.com/media-centre/edf-launches-britains-first-solar-tariff-no-standing-charge)**

The strongest usable evidence is that a suitable time-of-use arrangement may offer a three-hour 1am–4am charging window, and that charging from the grid overnight and discharging during expensive periods is the operating model relevant to a battery without solar. The source also indicates that 0% VAT and a 10-year warranty may apply to solar-and-battery installations, but it does not confirm that the same treatment or warranty applies to a standalone battery. Its £8,495 figure is for a complete solar package, so it must not be used as the price of a 5 kWh or 10 kWh battery-only system. Because the tariff is explicitly restricted to customers buying solar and a battery through EDF, it cannot support the conclusion that this tariff is available to the stated household without solar. The page provides no actual import or export rates, battery round-trip efficiency, maximum power, degradation assumptions, maintenance costs, replacement provisions or 2026 tariff terms. Consequently, it cannot calculate annual costs, payback or a 10-year net outcome for the four requested options. To complete that assessment, obtain current standalone-battery quotations, technical datasheets and warranty terms; half-hourly smart-meter consumption data; and current tariff prices and eligibility rules for a flat tariff and a battery-compatible time-of-use tariff. The unusually high daytime office demand is particularly important because electricity used directly during the day cannot normally be saved by a battery, reducing achievable cycling and savings.

**37. [S37] [Change or Renew Your Energy Tariff | New & Existing Customers | EDF](https://www.edfenergy.com/gas-and-electricity/change-energy-tariff)**

The page suggests that switching from a flat tariff to a suitable time-of-use tariff could produce meaningful savings even without solar, particularly if the household can use EDF’s approximately 6.49–6.99p/kWh overnight electricity between 11 pm and 6 am. The household’s high daytime office demand weakens the battery case: daytime consumption is often direct use rather than electricity that can be displaced by a battery, and the source provides no evidence that enough overnight energy would remain available to fill a 5 kWh or 10 kWh battery every day. A tariff-only switch is therefore the most plausible low-cost option to investigate first. The webpage does not support concluding that either battery size will pay for itself, nor does it provide the information needed to calculate annual costs, payback, degradation-adjusted savings, or a 10-year net result. Before deciding, obtain half-hourly smart-meter data, the proposed tariff’s complete unit-rate schedule and standing charges, confirmation that the tariff permits battery charging and export where relevant, and itemised quotations stating usable capacity, maximum charge/discharge power, efficiency, warranty, degradation or throughput guarantee, maintenance, replacement assumptions, and the VAT treatment of a battery-only installation. The strongest argument against buying the battery is that the office’s daytime load may be consumed while the battery is empty or may already be supplied directly, leaving insufficient peak-period energy to justify the capital cost; tariff rates may also change every three months.

**38. [S38] [Smart Export Guarantee & Tariff | Solar Energy | EDF](https://www.edfenergy.com/energy-efficiency/smart-export-tariff)**

This source does not establish that a battery-only installation would receive SEG payments: its FAQ ties eligibility to an eligible renewable generator, while the fixed 18p and 15p export products appear restricted, respectively, to qualifying battery-only arrangements under the terms and to solar PV. Without solar, the homeowner should not count export income unless EDF confirms in writing that grid-charged battery exports are eligible; the safest financial model assigns them zero SEG revenue. The source supports comparing a flat tariff with a time-of-use tariff, but it contains no import rates, so annual costs, payback, sensitivity analysis, and 10-year net outcomes for 5 kWh and 10 kWh batteries cannot be calculated from this webpage. On the supplied assumptions, the flat-rate cost is about £2,223/year before standing charges. A tariff-only switch could produce meaningful savings only if the off-peak rate is sufficiently below £0.247/kWh and the household can use enough electricity during those periods; the unusually high daytime office demand reduces that opportunity. A battery is financially worthwhile only if the usable energy shifted each year, after losses and degradation, multiplied by the import-price spread exceeds annualised installation, financing, maintenance and eventual replacement costs. The strongest argument against buying one is that the household’s demand is concentrated in daytime peak hours, while manual load shifting is limited, so a battery may cycle infrequently or discharge when its stored energy is not large enough to cover the office load. Before deciding, obtain half-hourly smart-meter data for at least 12 months, confirmed peak/off-peak import rates and standing charges, written confirmation of battery-only export eligibility, and comparable quotations specifying usable capacity, continuous and peak charge/discharge power, round-trip efficiency, warranty throughput and end-of-warranty capacity, installation and electrical-upgrade costs, maintenance, monitoring, emergency-backup capability, VAT treatment, and replacement assumptions.

**39. [S39] [Choosing the Best Energy Tariffs - How to pick the right one for you | EDF](https://www.edfenergy.com/energywise/choosing-best-energy-tariff)**

The page supports considering a smart time-of-use tariff because the household has a smart meter and could potentially charge a battery during cheaper periods. It also indicates that savings depend on when electricity is used; this is important because much of the household's unusually high demand occurs during daytime working hours and may already fall outside the battery's discharge window. The page provides no actual off-peak or peak prices, so it cannot establish annual costs, battery cycling, payback, or a 10-year net outcome for the 5 kWh and 10 kWh options. On the evidence supplied, switching to a suitable TOU tariff without a battery is the only option that can be assessed qualitatively and may provide meaningful savings where daytime or evening consumption can be moved to cheap periods. A battery's financial case requires current tariff quotations, installed prices, usable capacity, charge/discharge limits, efficiency, warranty and degradation terms, and the home's half-hourly consumption data. The strongest relevant implication is that a battery should not be assumed to cycle fully every day: the household must compare its overnight charging opportunity and peak-period demand with the battery's usable capacity, while recognising that daytime office use may reduce the amount of electricity available for profitable peak-time discharge.

**40. [S40] [July 2026 Energy Price Cap to rise by 13% | EDF](https://www.edfenergy.com/energywise/energy-price-cap-july-2026)**

The webpage indicates that electricity prices rose from 24.67p/kWh to 26.11p/kWh on 1 July 2026, with an electricity standing charge of about 57.19p/day, and it explicitly suggests comparing fixed or other tariffs rather than remaining on a standard variable tariff. For 9,000 kWh/year, the quoted July cap rates imply about £2,559/year including the quoted standing charge, before supplier discounts. This supports the conclusion that tariff switching could produce meaningful savings if a competitive tariff is below the cap, but the page supplies no suitable time-of-use prices or battery evidence. Consequently, it cannot determine whether either battery size pays for itself, calculate battery payback or 10-year net outcome, or assess the effects of daytime demand, cycling frequency, losses, degradation, installation cost, replacement, or tariff uncertainty. The homeowner would need actual half-hourly smart-meter consumption data, current flat and time-of-use tariff quotations, and itemised installed battery quotations specifying usable capacity, power limits, round-trip efficiency, warranty, degradation guarantee, maintenance, and replacement assumptions.

**41. [S41] [Should I fix my energy tariff before the July price cap? | Octopus Energy](https://octopus.energy/blog/should-I-fix-before-july-2026-price-cap/)**

For the stated household, the source supports considering a tariff change but not concluding that a 5 kWh or 10 kWh battery is financially worthwhile. At the existing flat electricity rate, annual consumption cost is approximately £2,223 before standing charges (9,000 × £0.247), but the webpage does not supply a comparable tariff rate. A tariff-only switch could produce meaningful savings if a suitable time-of-use tariff has a sufficiently large off-peak/peak spread and the household can use enough electricity overnight; however, the unusually high daytime office demand reduces the amount that can be shifted, and some overnight electricity may be consumed directly rather than stored. A battery would save only the spread captured on electricity that is actually charged, discharged, and used during high-price periods, less round-trip losses, degradation, financing or opportunity cost, and eventual replacement. Therefore no defensible annual saving, simple payback, 10-year net outcome, or sensitivity analysis for either battery size can be calculated from this webpage alone. The homeowner should obtain half-hourly smart-meter data, complete tariff quotations including standing charges and all time bands, and several installed battery quotations specifying usable capacity, maximum charge/discharge power, round-trip efficiency, warranty throughput or retained capacity, control software, VAT treatment, maintenance, and replacement assumptions. The strongest source-supported conclusion is to compare a competitive flat tariff with a suitable time-of-use tariff first; purchase a battery only if documented tariff spreads and the household’s measured load profile show enough recurring peak-period discharge to overcome the battery’s installed cost and degradation.

**42. [S42] [EDF launches first-of-its-kind three-rate tariff with free electricity hours | EDF](https://www.edfenergy.com/media-centre/edf-launches-first-its-kind-three-rate-tariff-free-electricity-hours)**

The source supports investigating a time-of-use tariff without a battery: the household already has a smart meter, and its 10–20 kWh/day office load would mostly occur in EDF’s cheaper 6am–4pm amber period, allowing savings without round-trip battery losses. The page’s maximum claimed saving is £187 per year versus EDF’s standard variable tariff, but that is not a guaranteed saving versus the household’s assumed competitive £0.247/kWh flat tariff and no exact FreePhase rates are supplied. A battery could arbitrage the 11pm–6am green period against the 4pm–7pm red period, but the source gives no red/green prices, usable capacity, charge/discharge limits, efficiency, degradation, warranty, installation cost or replacement cost. Therefore neither a 5 kWh nor a 10 kWh battery can be shown from this webpage to pay for itself; the likely best financial starting point is to obtain actual tariff rates and test the household’s half-hourly data against a tariff-only option. The conclusion would change only if the red–green price spread is consistently large, enough consumption occurs during 4pm–7pm, the battery is installed cheaply and retains capacity, and tariff rules remain favourable. The homeowner should obtain 12 months of half-hourly smart-meter data, the exact EDF red/amber/green rates and standing charges, competing tariff terms, and itemised quotations specifying usable capacity, maximum charge/discharge power, round-trip efficiency, warranty, degradation or throughput limits, maintenance, export/market-participation arrangements and replacement assumptions.

**43. [S43] [Electricity Tariffs: A Guide for Electric Car Drivers | EDF](https://www.edfenergy.com/electric-cars/tariffs-guide)**

For this household, the webpage supports checking a whole-home time-of-use tariff because a smart meter is required and off-peak pricing can reduce the cost of electricity shifted into the qualifying hours. Its strongest warning is that savings depend on when the home uses electricity and whether the cheap rate applies to the entire property; the household’s high daytime office demand would therefore limit savings if much of its consumption remains during the expensive period. The page does not establish that a battery is financially worthwhile and cannot support the requested annual-cost, payback, 10-year-net-outcome, or sensitivity calculations for 5 kWh versus 10 kWh storage. Those calculations require additional primary evidence: current comparable flat and time-of-use tariffs, standing charges, battery installed quotations, usable capacity, charge/discharge power, round-trip efficiency, degradation and warranty terms, replacement assumptions, export or flexibility payments, and actual half-hourly household consumption data.

**44. [S44] [Improving the energy efficiency of socially rented homes in England: government response  - GOV.UK](https://www.gov.uk/government/consultations/improving-the-energy-efficiency-of-socially-rented-homes-in-england/outcome/improving-the-energy-efficiency-of-socially-rented-homes-in-england-government-response)**

The page provides only general policy context: government identifies solar panels, insulation, and heat pumps as possible ways to reduce household energy bills, but it does not discuss batteries or electricity tariffs. Its MEES, VAT, grant, and £10,000 spend-exemption provisions concern social landlords and should not be applied to the homeowner in the question. No annual costs, savings, payback periods, 10-year outcomes, or sensitivity analysis can be calculated from this source. To answer the goal, evidence is still required from Ofgem, suppliers offering suitable time-of-use tariffs, battery manufacturers’ technical documentation, installer quotations, and independent battery-performance or consumer-trial data.

**45. [S45] [EDF's EV Tariffs For Your Car And Home | EV Tariffs | EDF](https://www.edfenergy.com/electric-cars/ev-tariffs)**

{
  "rational": "The webpage is directly relevant because it gives a current EDF smart-meter time-of-use tariff with a defined overnight charging window and off-peak prices, which can be used by a household without solar to charge a battery from the grid. It does not provide the tariff's peak rate, standalone home-battery eligibility, installed battery prices, usable capacity, charge/discharge limits, degradation, warranties, maintenance, VAT treatment, or replacement costs. Therefore, it suppor

**46. [S46] [EDF launches cheapest overnight charging on EV tariffs | EDF](https://www.edfenergy.com/media-centre/edf-launches-cheapest-overnight-charging-ev-tariffs)**

As of the page's 1 April 2026 information, EDF offers an unusually low 6.49–6.99p/kWh overnight rate from 11pm to 6am, which could improve the economics of tariff-only switching or battery charging. However, the source is an EDF EV-tariff announcement, not a home-battery assessment: it does not establish eligibility for a non-EV household, reveal the required peak price, or provide the technical and financial inputs needed to calculate annual costs, payback or a 10-year net outcome. The source therefore supports investigating a time-of-use tariff, but cannot show whether either a 5 kWh or 10 kWh battery would pay for itself. The homeowner should verify tariff eligibility and obtain the complete peak rate, standing charge and terms, then combine them with half-hourly consumption data and independent installed quotations before comparing no-battery and battery options.

**47. [S47] [Energy Price Cap: Everything You Need To Know | OVO Energy](https://www.ovoenergy.com/pricecap)**

{
  "rational": "The webpage is relevant for establishing the July–September 2026 price-cap context, confirming that actual costs depend on electricity consumption rather than the headline annual figure, and identifying smart meters and off-peak use as prerequisites or potential routes to savings. It does not provide battery prices, usable capacities, charge or discharge rates, degradation data, warranties, time-of-use tariff rates, or evidence about whether a battery would pay for itself. The

**48. [S48] [Earn free electricity on Sundays with our Sunday Saver Challenge | EDF](https://www.edfenergy.com/energy-efficiency/sunday-saver-challenge)**

This page supports the conclusion that a smart-metered household may obtain some savings by changing behaviour or using an EDF demand-shifting programme, but it is not evidence for the economics of installing a 5 kWh or 10 kWh battery. The household’s high daytime office consumption is especially important: electricity used during working hours cannot automatically benefit from a battery charged overnight unless it is deliberately shifted into the battery’s discharge period, and doing so incurs round-trip losses. The EDF offer is also not a conventional guaranteed time-of-use tariff: participation is monthly, challenges may not run, rewards are capped and changeable, and the relevant peak period is normally 4pm–7pm. Therefore, this source can inform option 2 and the value of limited manual load shifting, but additional primary evidence is required for indicative costs, payback, degradation, replacement risk, and the 10-year comparison of flat tariff, time-of-use tariff, 5 kWh battery, and 10 kWh battery.

**49. [S49] [Overnight EV Tariff For Your Car And Home | GoElectric | EDF](https://www.edfenergy.com/electric-cars/ev-tariffs/goelectric)**

The page supports the conclusion that a suitable overnight time-of-use tariff could produce meaningful savings without a battery if the household can consume a substantial share of its electricity between 11 pm and 6 am. At the stated 6.99p/kWh off-peak rate, the gross saving versus the household’s 24.7p/kWh flat rate is approximately 17.71p/kWh before standing charges and any peak-rate premium. However, the household’s unusually high daytime office demand means much of its consumption may not qualify for the cheap period, and a battery would incur charging losses and capital costs. EDF’s own 6% threshold is only a supplier rule of thumb, not proof of profitability for this household. The tariff cannot be recommended on this source alone because it is EV-only, and the webpage contains no evidence needed to calculate 5 kWh or 10 kWh battery payback or a 10-year net outcome. Before deciding, the homeowner should obtain half-hourly smart-meter data, confirm eligibility and the complete peak/off-peak tariff including standing charges, and obtain independent installed quotations specifying usable capacity, continuous charge/discharge power, round-trip efficiency, degradation warranty, cycle or throughput limits, replacement coverage, export/import controls, and backup capability.

**50. [S50] [Boiler Upgrade Scheme (BUS) - Property owners | Ofgem](https://www.ofgem.gov.uk/environmental-and-social-schemes/boiler-upgrade-scheme-bus/boiler-upgrade-scheme-bus-property-owners)**

This source provides no usable evidence for comparing a flat-rate tariff, a time-of-use tariff, or 5 kWh and 10 kWh batteries. Its relevant conclusion is limited: the Boiler Upgrade Scheme is for qualifying heating technologies, not standalone batteries, and the quoted 0% VAT information is explicitly about energy-saving-material installations such as heat pumps and biomass boilers, not battery storage. A proper financial assessment would require separate primary or independent sources for current tariffs, battery installation quotations, usable capacity, charge/discharge limits, round-trip efficiency, degradation and warranty terms, replacement costs, and applicable electricity-market rules.

**51. [S51] [bakewell-town-guide-2025.pdf](https://www.bakewelltowncouncil.gov.uk/uploads/bakewell-town-guide-2025.pdf?v=1750841577)**

No reliable evidence can be extracted from this source to calculate annual electricity costs, savings, payback, or a 10-year outcome for the four requested options. A searchable PDF, HTML page, OCR text, or the specific relevant sections would be needed. Without that material, any numerical conclusion would rely on external assumptions rather than the provided webpage.

**52. [S52] [The UK's Integrated National Energy and Climate Plan - GOV.UK](https://assets.publishing.service.gov.uk/media/60bdd2d2e90e0743ae8c284e/uk-integrated-national-energy-climate-plan-necp-31-january-2020.pdf)**

This source cannot support the requested comparison of a flat-rate tariff, a time-of-use tariff, or 5 kWh and 10 kWh batteries. A readable webpage or PDF text layer is required before extracting evidence or calculating annual costs, savings, payback, and 10-year outcomes. The requested assessment should not be fabricated from this unreadable source.

**53. [S53] [Clean flexibility roadmap: July 2026 update (accessible webpage) - GOV.UK](https://www.gov.uk/government/publications/clean-flexibility-roadmap/clean-flexibility-roadmap-july-2026-update-accessible-webpage)**

This source supports switching to a suitable time-of-use tariff as a potentially worthwhile low-cost first step: the household has a smart meter, and the government's stated mechanism is to move consumption away from expensive peak periods or toward cheaper periods. However, the household's unusually high daytime office demand limits the amount of load that can be shifted to overnight periods, so the source alone cannot establish that a battery would add enough arbitrage value to recover its installed cost. The policy reference to possible future removal of levies and improved access to flexibility markets is also a reason to treat battery returns as uncertain rather than guaranteed. For the requested comparison, this webpage should therefore be used as policy context only; tariff quotations, half-hourly consumption data, and technical and installed-cost quotations from manufacturers or installers are still required to calculate annual costs, payback, and the 10-year outcome.

**54. [S54] [Clean flexibility roadmap (accessible webpage) - GOV.UK](https://www.gov.uk/government/publications/clean-flexibility-roadmap/clean-flexibility-roadmap)**

The webpage supports the general proposition that a smart meter, time-of-use tariff and flexible battery operation can reduce electricity costs, especially where consumption can be moved into cheap periods. It also cautions that savings depend on genuine demand flexibility, which weakens the case for a battery when much consumption occurs during daytime working hours and only a small amount can be shifted manually. The source does not provide the evidence needed for the requested numerical comparison of flat tariff, time-of-use tariff, 5 kWh battery and 10 kWh battery. To complete that assessment, additional current sources and quotations are required for English retail tariff prices and peak/off-peak windows, installed battery costs and VAT treatment, usable capacity, charge/discharge limits, round-trip efficiency, degradation and warranty terms, maintenance and replacement costs, export or flexibility-service payments, and applicable Ofgem, government and electricity-market rules. On this webpage alone, changing to a suitable time-of-use tariff may provide meaningful savings if enough demand can be placed in cheap periods; the financial case for either battery size remains unproven, and the unusually high daytime demand is a material argument against assuming frequent full battery cycles.

**55. [S55] [Feed-in Tariffs (FIT) | Ofgem](https://www.ofgem.gov.uk/environmental-and-social-schemes/feed-tariffs-fit)**

This source supports one limited conclusion: a new standalone home battery in England should not be assumed to receive Feed-in Tariff payments, because the FIT scheme closed to new applicants on 1 April 2019 and was designed around accredited electricity-generating and exporting installations. It does not support an annual-cost, payback, or 10-year comparison between flat-rate electricity, a time-of-use tariff, and 5 kWh or 10 kWh batteries. Those calculations require current supplier tariff schedules and battery quotations, plus technical and financial assumptions from manufacturers or independent sources. On the evidence supplied, no conclusion can be drawn about whether either battery size pays for itself; the strongest source-based finding is that FIT is not a relevant new-battery revenue stream.

**56. [S56] [Smart Secure Electricity Systems (SSES) Programme: first phase energy smart appliances regulations - consultation document (accessible webpage) - GOV.UK](https://www.gov.uk/government/consultations/smart-secure-electricity-systems-sses-programme-first-phase-energy-smart-appliances-regulations/smart-secure-electricity-systems-sses-programme-first-phase-energy-smart-appliances-regulations-consultation-document-accessible-webpage)**

This source supports a cautious preliminary conclusion: switching from a flat tariff to a genuinely suitable time-of-use tariff may provide meaningful savings even without a battery, with the government citing savings of over £200 per year for households that avoid peak periods. However, that figure is conditional, not battery-specific, and may not fit a home with high daytime demand unless the tariff's cheap period overlaps with the office's consumption. The source does not justify assuming daily full battery cycling or calculating payback for either a 5 kWh or 10 kWh battery. A battery's financial case would require actual half-hourly consumption, the supplier's complete time-of-use rate structure and export/charging rules, battery usable capacity and charge/discharge limits, efficiency, degradation and warranty terms, and firm installed quotations. The proposed smart-appliance rules may improve measurement, cybersecurity, and grid-stability standards, but they do not provide a subsidy or guarantee that a battery will pay for itself. Source: GOV.UK, “Smart Secure Electricity Systems (SSES) Programme: first phase energy smart appliances regulations – consultation document,” issued 1 December 2025, https://www.gov.uk/government/consultations/smart-secure-electricity-systems-sses-programme-first-phase-energy-smart-appliances-regulations/smart-secure-electricity-systems-sses-programme-first-phase-energy-smart-appliances-regulations-consultation-document-accessible-webpage

**57. [S57] [Assessing the case for community batteries: call for evidence](https://assets.publishing.service.gov.uk/media/6a2041a058aae59498cb29e5/assessing-the-case-for-community-batteries-call-for-evidence.pdf)**

No usable evidence can be extracted from the provided webpage, so it is not possible to calculate indicative annual costs, savings, payback, 10-year outcomes, or sensitivity analysis for the four options. A readable webpage, PDF text layer, URL, or separate source documents containing current UK tariff rates, installed quotations, and technical specifications are required before making the assessment.

**58. [S58] [Smart Secure Electricity Systems (SSES) Programme: Enduring Governance (accessible webpage) - GOV.UK](https://www.gov.uk/government/consultations/smart-secure-electricity-systems-programme-sses-enduring-governance/smart-secure-electricity-systems-sses-programme-enduring-governance-accessible-webpage)**

The strongest evidence from this page supports considering a time-of-use tariff even without a battery: the government says that some households can save over £200 annually by moving from a price-cap-style tariff to a market-linked tariff, particularly where consumption is already outside the 4pm–7pm peak. However, the stated example is not directly transferable to this household because much of its office demand occurs during daytime working hours, and the source gives no tariff prices or battery economics. For the stated comparison, the page supports option 2 as a potentially meaningful low-cost intervention, while it cannot demonstrate that either a 5 kWh or 10 kWh battery will pay for itself. A battery’s value would depend on the actual off-peak/peak price spread, how much electricity can be shifted without a full daily cycle, usable capacity and power limits, round-trip losses, degradation, installed cost, replacement risk, and tariff availability. No annual costs, payback periods, 10-year outcomes, or sensitivity results can be calculated reliably from this webpage alone; those require current supplier tariffs, quotations, technical datasheets, and interval-meter consumption data.

**59. [S59] [Zero VAT on Home Battery Storage: The HMRC Rules, Real Savings, and What to Check Before March 2027 | Mackie Electrical](https://www.mackie-electrical.co.uk/zero-vat-battery-storage-guide)**

The webpage supports three conclusions relevant to the goal: solar panels are not required for UK domestic battery VAT relief during the stated window; a battery’s financial value mainly comes from charging on a cheap time-of-use tariff and discharging when electricity is expensive; and indicative installer estimates are roughly £4,500–£5,500 before VAT and £250–£350 annual arbitrage savings for 5 kWh, versus £6,000–£7,500 before VAT and £400–£650 for about 9.5 kWh. The source’s worked example assumes a 20p/kWh peak-to-off-peak spread and 10% losses, but it is based on Southern Scotland Flux prices and assumes substantial use of the battery during peak and standard periods. For the described England household, unusually high daytime office demand may reduce savings because much consumption occurs before the evening peak, while limited ability to shift demand may prevent full cycling. The page therefore suggests that tariff switching without a battery could provide meaningful savings if the household can move enough consumption to cheap periods, whereas battery payback is uncertain and likely near the length of a typical warranty unless installed cost is low, spreads remain large, and the battery is regularly used. A proper decision requires England-specific tariff data, half-hourly consumption, quoted usable capacity and power limits, warranty/degradation terms, and competing installed quotations.

**60. [S60] [Battery Storage Without Solar UK 2026: Cost & Payback Guide | HeatPumpsAndSolar](https://heatpumpsandsolar.co.uk/insights/battery-storage-without-solar-uk)**

{
  "rational": "The webpage directly addresses whether a UK home can use a battery charged from the grid, rather than solar, and provides relevant 2026 tariff assumptions, installed-price ranges, degradation estimates, warranty information, VAT treatment, and grid-connection requirements. It is particularly relevant to this household because it warns that savings depend on actual evening consumption rather than battery capacity. However, it does not provide a sufficiently detailed load profile 

**61. [S61] [Solar Panels Scotland | Free Quotes from Accredited Installers | Scottish Energy Efficiency](https://scottishenergyefficiency.co.uk/battery-storage-scotland)**

The only potentially useful evidence is the advertised £2,900 price for one 5.32 kWh Duracell battery and the webpage’s statement that batteries were subject to 0% VAT in the stated period, but both require verification for an English installation and the price may not be fully installed. The page offers no suitable time-of-use tariff, peak/off-peak prices, charge or discharge limits, usable capacity, round-trip efficiency, degradation rate, warranty terms, replacement assumptions, or evidence about cycling a battery for a high-daytime-demand household. Consequently, it cannot support indicative annual costs, payback, 10-year outcomes, sensitivity analysis, or a conclusion that either a 5 kWh or 10 kWh battery is worthwhile without solar. Its solar-plus-battery savings and Scottish payback claims should be excluded from the requested England battery-only comparison.

</details>
