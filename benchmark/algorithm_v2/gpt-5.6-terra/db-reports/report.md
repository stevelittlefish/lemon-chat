---

## Research Summary

**Duration:** 373.0s | **Rounds:** 6 | **Queries:** 19 | **URLs analyzed:** 38 | **Model:** openai/gpt-5.6-terra | **Category:** Comparison

---

# Is a Home Battery Worthwhile Without Solar Panels for a UK Home?

## Executive summary

For the household described—an owner-occupied home in England using around 9,000 kWh of electricity annually, with high daytime home-office demand and no planned solar installation—the strongest financial move is likely to be **changing electricity tariff before buying a battery**.

A suitable time-of-use (ToU) tariff could save approximately **£330 per year** compared with remaining on a 24.7p/kWh flat-rate tariff, even without a battery. That result depends on obtaining a tariff whose daytime rate is not punitive and whose expensive period is concentrated in the early evening, when the household can reduce exposure naturally or through modest behavioural changes.

A battery can add savings by charging overnight and discharging during expensive periods. However, under realistic assumptions about usable capacity, charging losses, battery power limits, standby consumption, imperfect utilisation, degradation and installation costs:

- A roughly **5 kWh battery** is likely to save only around **£205–£210 in its first year** beyond the ToU tariff alone. At an installed cost around **£4,050**, simple payback is roughly **19–20 years**.
- A roughly **10 kWh battery** is likely to save roughly **£205–£220 in its first year** beyond ToU alone, but typically costs around **£6,150**. It is less likely to be fully used and has a simple payback close to **30 years**.
- Over ten years, neither system is likely to recover its installed cost from electricity-price arbitrage alone.

The recommended sequence is therefore:

1. Obtain and analyse twelve months of half-hourly smart-meter data;
2. Compare postcode-specific flat and ToU tariffs;
3. Switch tariff without buying a battery;
4. Consider a small battery only if the data show consistently high 16:00–19:00 imports, the tariff permits battery operation, and an unusually low fully installed quotation is available.

The major caveat is that future smart tariffs could offer substantially wider and more reliable price spreads. If cheap overnight electricity is around 7–9p/kWh while electricity reliably avoided by the battery costs 30–35p/kWh or more, a small battery could become more attractive. But that tariff environment cannot safely be assumed for a ten-year investment.

---

## Assessment basis and key assumptions

This report assesses the position as at **14 July 2026**. The household uses approximately 9,000 kWh per year, or around **24.7 kWh per day**. The existing electricity unit rate is approximately **24.7p/kWh**, which is relatively competitive against the July–September 2026 average price-cap benchmark of approximately 26.11p/kWh plus a 57.19p/day standing charge. Actual regional rates differ, but the household should not assume that simply moving to a standard variable tariff would improve its position. [S17]

The flat-tariff baseline assumes a standing charge of approximately **55p/day**, or around £201 per year. The current flat-rate annual bill is therefore modelled as:

\[
9,000 \text{ kWh} \times £0.247 + £201 = approximately £2,424/year
\]

The household has a smart meter, which is important because smart tariffs normally require half-hourly consumption data. EDF and E.ON both state that smart-meter data is required for relevant time-of-use products. [S20] [S28]

The household’s home office is particularly important. It consumes approximately 10–20 kWh per day, much of it during working hours. That means there is a large amount of electricity demand which cannot simply be moved manually to midnight. However, the key issue for a battery is not total daytime demand: it is the amount of demand occurring during the tariff’s most expensive periods, usually around **16:00–19:00**. EDF’s three-rate tariff structure, for example, places the high-price period in that three-hour evening window, while much daytime consumption falls outside it. [S30]

### Indicative consumption profile

Without the household’s actual half-hourly data, the following profile is used for modelling:

| Period | Share of annual use | Annual electricity demand |
|---|---:|---:|
| Overnight off-peak period | 12% | 1,080 kWh |
| Evening peak, 16:00–19:00 | 20% | 1,800 kWh |
| Other daytime and evening hours | 68% | 6,120 kWh |
| **Total** | **100%** | **9,000 kWh** |

The 12% overnight demand is assumed to be used directly by appliances such as refrigeration, broadband equipment, charging, kitchen appliances and background load. It is not incorrectly treated as electricity that must be put through the battery. This distinction matters: direct overnight electricity receives the cheap tariff rate without suffering battery losses.

---

## Suitable tariffs and tariff uncertainty

The most tempting advertised overnight tariffs are often EV tariffs. EDF has advertised seven-hour overnight rates of around 6.49–6.99p/kWh on EV products, but these tariffs are explicitly presented as EV-related and should not be assumed available to a household with no qualifying EV. [S5] [S22] Similarly, Octopus Go and Intelligent Octopus Go generally require a qualifying EV or plug-in hybrid. [S21] E.ON’s low-rate Drive products are likewise EV-oriented unless the supplier confirms otherwise in writing. [S18]

The model therefore does **not** assume access to a 6–9p/kWh EV rate. Instead, it uses an illustrative three-rate tariff structure broadly consistent with published EDF-style tariff information:

| Tariff period | Assumed unit rate |
|---|---:|
| Off-peak: 01:00–04:00 | 10.361p/kWh |
| Normal daytime/evening | 20.22p/kWh |
| Peak: 16:00–19:00 | 30.22p/kWh |
| Standing charge | 55p/day |

These rates are illustrative rather than guaranteed. EDF confirms that it has operated a three-period structure with the expensive period limited to 16:00–19:00, a cheaper overnight period, and intermediate daytime/evening pricing. [S30] The specific 10.361p, 20.22p and 30.22p rates should be treated cautiously because they originate from secondary tariff comparison material rather than a durable supplier tariff label. [S4]

Tariff uncertainty is central to the battery decision. Octopus warns that smart-tariff pricing can change and that historical dynamic prices do not predict future prices. [S21] A battery is a ten-year-plus capital asset, while a tariff can be withdrawn, repriced, restricted to particular equipment, or altered in its cheap hours within months.

---

# Comparison Table

## Comparison Table

| Criterion | 1. Flat tariff, no battery | 2. ToU tariff, no battery | 3. ToU + roughly 5 kWh battery | 4. ToU + roughly 10 kWh battery |
|---|---|---|---|---|
| Upfront capital cost | £0 | £0 | About £4,050 | About £6,150 |
| Estimated annual bill at 9,000 kWh | **£2,424** | **£2,094** | **£1,885–£1,890** | **£1,885–£1,890** |
| Annual saving versus flat tariff | — | About **£330** | About **£535–£540** | About **£535–£540** |
| Additional saving versus ToU only | — | — | About **£205–£210** | About **£205–£220** |
| Simple payback | n/a | Immediate | About **19–20 years** | About **28–30 years** |
| Ten-year financial result versus flat tariff | Baseline | About **+£3,300** | About **+£1,100** after battery cost | About **–£1,000** after battery cost |
| Financial risk | Low | Moderate tariff risk | High tariff and equipment risk | High tariff and equipment risk |
| Ability to reduce 16:00–19:00 imports | No | Limited | Moderate | Moderate, but often under-used |
| Backup potential | None | None | Possible with extra hardware | Possible with extra hardware |
| Likely best use case | Maximum simplicity | Best financial value | Cheap quote plus strong evening peak load | Solar household, EV household, or much larger peak demand |
| Overall rating for this household | ✓ Acceptable baseline | ✓✓✓ Recommended | ✓ Only in favourable conditions | ✗ Not recommended for arbitrage alone |

---

# Option 1: Remain on a competitive flat-rate tariff without a battery

## Cost and financial outcome

At 9,000 kWh per year and 24.7p/kWh, electricity consumption costs approximately £2,223 annually. Adding an assumed £201 annual standing charge produces a bill of approximately **£2,424 per year**.

This option involves no equipment cost, no tariff complexity and no risk of battery failure. It also avoids the need to monitor half-hourly consumption or change household behaviour around peak periods.

## Strengths

The main strength is certainty. The household already has a relatively competitive unit price compared with the July 2026 average capped variable electricity rate of around 26.11p/kWh. [S17] If the existing tariff is fixed or otherwise competitively priced, moving hastily to a high-daytime-rate ToU tariff could make the household worse off.

A flat tariff also suits a home office with inflexible daytime consumption. If computers, screens, networking equipment, servers, heating or cooling systems and office appliances must run during working hours, the household does not need to worry whether those hours are “green,” “amber” or “peak.”

## Weaknesses

The weakness is that the household pays the same price whether electricity is used at 03:00 or 18:00. It cannot benefit from cheap overnight electricity or avoid expensive grid periods.

This option also leaves approximately £330 per year of plausible tariff-only savings unclaimed under the central ToU scenario. That is meaningful, particularly because it requires no capital expenditure.

## Ideal use case

This option is best for a household that values simplicity, cannot tolerate bill volatility, or cannot find a ToU tariff with rates that suit its actual half-hourly usage.

---

# Option 2: Switch to a suitable time-of-use tariff without a battery

## Cost and financial outcome

Using the assumed consumption profile and three-rate tariff:

- Overnight: 1,080 kWh × 10.361p = approximately £112
- Peak 16:00–19:00: 1,800 kWh × 30.22p = approximately £544
- Other hours: 6,120 kWh × 20.22p = approximately £1,237
- Standing charge: approximately £201

Total annual cost is approximately **£2,094**, a saving of approximately **£330 per year** versus the flat tariff.

That saving arises despite only 12% of consumption occurring overnight. The key is that most daytime office consumption is assumed to occur at the normal 20.22p/kWh rate rather than the 30.22p/kWh evening peak rate. A tariff with a limited evening peak can therefore suit a home-office household better than a conventional Economy 7 tariff with expensive daytime electricity.

## Strengths

This is the strongest financial option because it creates savings without capital risk. There is no battery to degrade, no inverter to fail, no installation disruption and no concern that the equipment will become obsolete before it pays back.

E.ON has described smart tariff structures where lower-priced periods can extend through much of the daytime, such as 05:00–16:00, potentially fitting home-office demand better than a simple overnight-only tariff. [S16] [S31] The exact rates must be checked, but the principle is important: the best tariff is not necessarily the tariff with the cheapest overnight window. It is the tariff that produces the lowest total annual cost for the home’s actual load profile.

## Weaknesses

The risk is choosing the wrong tariff. Standard Economy 7-style pricing can have cheap overnight electricity but materially more expensive daytime rates. A household that consumes much of its electricity during office hours may pay more overall if it cannot shift enough usage. E.ON specifically notes the relevance of usage timing to time-of-use tariff outcomes. [S12] [S28]

The household should also avoid relying on promotional “free electricity” events. EDF states that such events are not guaranteed and depend on grid conditions and location. [S13]

## Ideal use case

This is the ideal choice for this household if its half-hourly data show that a meaningful share of use is outside the 16:00–19:00 peak and if a postcode-specific tariff quote produces a lower annual cost than the existing flat tariff.

---

# Option 3: Install a roughly 5 kWh battery and switch tariff

## Battery assumptions

A representative small system is modelled with:

| Assumption | Value |
|---|---:|
| Nominal capacity | 5.2 kWh |
| Assumed usable capacity | 4.68 kWh |
| Continuous discharge power | 2.6 kW |
| Round-trip efficiency | 90% |
| Installed cost | £4,050 |
| Standby consumption | 88 kWh/year |
| Capacity remaining in year 10 | 80% of initial usable capacity |

Indicative market material suggests small battery systems around this size can cost roughly £3,800–£4,300 installed, though quotes vary materially with inverter specification, electrical work, backup hardware, access difficulty and region. [S10] Secondary installer and comparison sources also place typical small-system installed costs near £4,000. [S9]

The 2.6 kW discharge limit matters. If household load reaches 4 kW during the early evening—for example, office loads, cooking, laundry and other domestic demand—the battery can serve only around 2.6 kW. The remainder is still imported at the peak tariff rate.

## Why the battery is not assumed to cycle fully every day

A 4.68 kWh usable battery could theoretically deliver close to that amount daily. But it should not be assumed to do so 365 days per year. The battery must have sufficient stored energy, there must be enough simultaneous household load when prices are high, and the tariff must provide a sufficiently attractive charging period.

The central model assumes **1,400 kWh delivered annually**, equivalent to about **299 equivalent full cycles per year**. This is deliberately below one full cycle every day.

Of the delivered electricity, 75% is assumed to replace peak imports and 25% to replace normal-rate electricity. That gives an average avoided import cost of:

\[
(75\% \times 30.22p) + (25\% \times 20.22p) = 27.72p/kWh
\]

At 90% round-trip efficiency, delivering 1 kWh requires approximately 1.111 kWh of off-peak electricity. The effective charging cost is therefore:

\[
10.361p \div 0.90 = 11.51p/kWh\ delivered
\]

The gross arbitrage margin is:

\[
27.72p - 11.51p = 16.21p/kWh
\]

Annual gross arbitrage benefit:

\[
1,400 \times 16.21p = approximately £227
\]

After allowing for roughly £18 per year of standby electricity, the first-year net battery benefit is approximately **£209**, rounded to **£205–£210**.

## Cost, payback and ten-year outcome

The battery reduces the ToU-only annual bill from approximately £2,094 to around **£1,885–£1,890**. That is approximately £535–£540 lower than the flat-tariff baseline, but only around £205–£210 lower than simply changing tariff without buying a battery.

At a £4,050 installed cost:

\[
£4,050 \div £209 \approx 19.4\ years
\]

That is before allowing for battery degradation, inverter failure, financing costs or tariff changes.

Assuming battery capacity declines linearly to 80% of its initial usable capacity by year ten, the ten-year battery-arbitrage benefit is approximately **£1,850–£1,900**. Adding the £3,300 ten-year tariff saving but subtracting the £4,050 purchase cost produces a ten-year financial result of about:

\[
£3,300 + £1,875 - £4,050 = approximately +£1,125
\]

That is still better than remaining on the flat tariff, but materially worse than simply switching tariff and keeping the £4,050 capital.

## Ideal use case

A 5 kWh battery is potentially defensible only if all of the following apply:

- Half-hourly data show large and regular 16:00–19:00 imports;
- The battery’s 2.6 kW discharge limit can cover a useful share of those loads;
- The household obtains a battery-compatible tariff with a reliable wide price spread;
- The installed price is unusually low, ideally closer to £2,000 than £4,000;
- Backup power has genuine additional value to the homeowner.

---

# Option 4: Install a roughly 10 kWh battery and switch tariff

## Battery assumptions

The larger system is modelled with:

| Assumption | Value |
|---|---:|
| Nominal capacity | 9.5 kWh |
| Assumed usable capacity | 8.55 kWh |
| Continuous discharge power | 3.6 kW |
| Round-trip efficiency | 90% |
| Installed cost | £6,150 |
| Standby consumption | 88 kWh/year |
| Capacity remaining in year 10 | 80% of original usable capacity |

Indicative pricing for a system around this size is roughly £5,800–£6,500 installed. [S10] Those are not guaranteed market prices: a detailed quote should identify the battery, inverter, isolation equipment, consumer-unit work, monitoring platform, DNO work, VAT treatment and whether backup hardware is included.

## Why bigger is not necessarily better

The 10 kWh battery has more capacity, but the household may not have enough expensive-period demand to use it. The key tariff peak is assumed to last only three hours, from 16:00–19:00. At 3.6 kW, the battery could theoretically deliver 10.8 kWh over three hours, but only if the household is continuously drawing at least that much power and the battery began the period fully charged.

That is unlikely every day. The household has high daytime office demand, but much of that demand may occur before 16:00, when the normal tariff rate is 20.22p/kWh rather than the 30.22p/kWh peak rate. Extra battery capacity may therefore displace lower-value normal-rate electricity rather than expensive peak electricity.

The central model assumes 1,580 kWh of delivered energy annually, or approximately 185 equivalent full cycles. This is notably lower utilisation than the 5 kWh system.

## Corrected central financial result

If 55% of the battery’s output replaces peak imports and 45% replaces normal-rate imports, the average displaced import cost is:

\[
(55\% \times 30.22p) + (45\% \times 20.22p) = 25.72p/kWh
\]

That yields a gross margin of:

\[
25.72p - 11.51p = 14.21p/kWh
\]

At 1,580 kWh delivered annually, gross benefit is approximately £225. After £18 of standby electricity, the first-year net benefit is about **£206**.

This is an important conclusion: the larger battery’s extra capacity does not automatically produce higher savings. Its annual arbitrage benefit is similar to the small battery’s, because it is less fully used and displaces cheaper electricity more often.

The simple payback is therefore around:

\[
£6,150 \div £206 \approx 30\ years
\]

Over ten years, after degradation, the battery may produce around £1,800–£1,900 of arbitrage benefit. Against a £6,150 installed cost, the household is likely to be around **£1,000 worse off over ten years than if it simply switched to the ToU tariff without buying a battery**.

## Ideal use case

A 10 kWh battery is generally not justified for this household without solar. It becomes more plausible if the home later adds solar panels, an EV, a heat pump, electric heating, frequent high-power evening demand, or a strong need for backup resilience. But for import-price arbitrage alone, it is oversized.

---

## Sensitivity analysis

### Annual consumption sensitivity

The result changes with total consumption, but not as much as might be expected. A battery is limited by its own usable capacity, power limit and the number of valuable cycles it can complete—not simply by the household’s total annual consumption.

| Annual household use | ToU-only saving versus flat | 5 kWh extra saving versus ToU | 10 kWh extra saving versus ToU |
|---|---:|---:|---:|
| 6,000 kWh/year | About £220/year | About £140/year | About £150/year |
| 9,000 kWh/year | About £330/year | About £205–£210/year | About £205–£220/year |
| 11,000 kWh/year | About £400/year | About £220/year | About £230/year |

At 6,000 kWh per year, the household is less likely to have enough high-price demand to use the battery regularly. At 11,000 kWh, there is more demand available, but the battery remains constrained by its capacity and power rating.

### Tariff spread sensitivity

The key formula is:

\[
Battery\ benefit = delivered\ kWh \times (avoided\ import\ price - off\ peak\ price / efficiency)
\]

For the 5 kWh system, assume 1,400 kWh delivered annually.

| Scenario | Cheap charging price | Average avoided import price | Gross annual margin | Approx. net annual benefit after standby |
|---|---:|---:|---:|---:|
| Narrow spread | 14p/kWh | 25p/kWh | 9.44p/kWh | About £114 |
| Central case | 10.361p/kWh | 27.72p/kWh | 16.21p/kWh | About £209 |
| Wide spread | 9p/kWh | 35p/kWh | 25p/kWh | About £332 |

The wide-spread scenario is attractive, but it should not be treated as a safe forecast. EV tariffs may be unavailable, smart tariff structures may change, and high daytime prices can make the tariff expensive if the battery does not discharge as planned.

### Peak-load sensitivity

For the 5 kWh system, central modelling assumes 75% of output replaces peak-rate electricity. If all 1,400 kWh replaced peak-rate imports, the annual benefit improves by only around **£35** relative to the 75% case, because the difference between the assumed peak and normal rates is 10p/kWh and only 25% of output changes category.

The larger improvement occurs when peak displacement rises from around 50% to 100%: that can be worth around **£70 per year** for 1,400 kWh of output. This shows why half-hourly load data is more valuable than generic annual consumption figures.

### Installation-cost sensitivity

For the 5 kWh battery to compete with ToU-only over ten years under the central assumptions, its installed cost would need to be close to its expected ten-year arbitrage benefit: approximately **£1,850–£1,900**, perhaps around £2,000 if the homeowner values modest backup capability.

That is far below the indicative £3,800–£4,300 installed range. [S10]

A £6,150 10 kWh system would need either dramatically greater annual utilisation, much wider tariff spreads, significant solar self-consumption benefits, or a substantial non-financial value assigned to backup power.

### Degradation and replacement sensitivity

The central model assumes capacity falls to 80% by year ten. Indicative warranty information commonly protects capacity only down to a lower retained-capacity threshold, such as 70%, and may not cover all labour or associated equipment costs. [S10]

If the battery declines to 70% rather than 80% by year ten, the 5 kWh system’s ten-year energy-arbitrage value falls by roughly £100–£150. If the inverter requires replacement outside warranty, the economics worsen materially.

No major repair or replacement cost has been included in the central calculation. That omission is favourable to the battery case.

---

## Shared Considerations

### Battery warranties, power and maintenance

A battery warranty is not the same as a promise of a ten-year financial return. Home batteries commonly have capacity-retention limits, cycle or throughput conditions, exclusions and varying labour cover. The homeowner should request the exact warranty documents rather than relying on an installer’s summary.

Power matters as much as capacity. A 5 kWh battery with a 2.6 kW inverter cannot cover a 4–6 kW household peak. A 10 kWh battery with a 3.6 kW inverter still cannot serve every simultaneous appliance load. [S10]

Maintenance is generally limited, but “low maintenance” is not “zero cost.” The owner may face monitoring subscriptions, inverter replacement, firmware problems, installer call-out charges or changes to supplier integrations.

### Backup power is not automatic

A grid-connected battery does not necessarily power the house during an outage. Backup operation may require a dedicated backup gateway, critical-load consumer unit, isolation equipment and installation work. Those additions can raise cost materially. Secondary pricing evidence specifically warns that inverters and backup hardware can add significant cost. [S9]

For this household, backup is a secondary benefit. It should therefore not be used to justify a battery unless the homeowner puts a realistic monetary value on avoiding outages.

### VAT, grants, export and grid rules

No general battery-only grant has been included in this assessment. The VAT position for a standalone battery installation should be confirmed in writing by the installer and checked against current HMRC treatment. EDF’s solar tariff material confirms zero-rate VAT treatment in relevant solar energy-saving-material contexts, but does not establish that every standalone battery installation receives equivalent treatment. [S27]

Smart Export Guarantee income has been excluded. SEG is designed around export from qualifying renewable generation, not simply importing cheap grid electricity, storing it and exporting it later. [S26] [S33] A homeowner should not assume that grid-charged battery export is permitted or paid without written supplier confirmation.

The installer should also confirm the relevant Distribution Network Operator notification or approval process, export limitation arrangement and whether the system is configured to prevent unintended grid export.

### Why manufacturer savings claims should be treated cautiously

GivEnergy promotes battery-only savings and a five-year return, but the published claim lacks the necessary assumptions on battery size, tariff rates, degradation, cycling, standing charges and actual household demand. [S25] Such claims may reflect favourable tariffs, high cycling, solar generation, export revenue or customer profiles unlike this household.

Similarly, supplier examples involving solar and batteries are not transferable to a battery-only household. EDF’s illustrative battery savings material includes solar generation, import/export arrangements and solar self-consumption. [S27]

---

## What data and quotations to obtain before buying

Before making a purchase, the homeowner should obtain the following.

1. **Twelve months of half-hourly smart-meter import data.** Analyse:
   - kWh imported from 16:00–19:00;
   - overnight demand;
   - average and maximum household demand;
   - weekday versus weekend patterns;
   - seasonal differences;
   - the home office’s actual contribution to peak demand.

2. **At least three postcode-specific electricity tariff illustrations**, including:
   - all unit rates by time period;
   - standing charge;
   - tariff duration and exit fee;
   - smart-meter and half-hourly data requirements;
   - battery eligibility;
   - whether grid charging is allowed;
   - whether export is permitted or restricted;
   - whether supplier automation is required.

3. **Three itemised battery quotations** that separate:
   - battery modules;
   - inverter;
   - installation labour;
   - consumer-unit or electrical upgrades;
   - DNO work;
   - monitoring;
   - backup equipment;
   - VAT;
   - export limitation;
   - scaffolding or access costs if any.

4. **Technical documents**, not just sales brochures:
   - usable capacity, not only nominal capacity;
   - continuous charge and discharge power;
   - round-trip efficiency;
   - standby consumption;
   - warranty throughput limit;
   - guaranteed retained capacity;
   - inverter warranty;
   - labour warranty;
   - replacement costs after warranty.

---

## Best For verdicts

**Best for financial return: Option 2, a suitable ToU tariff without a battery.** It offers approximately £330 per year of savings in the central case with no upfront capital cost and no battery-performance risk.

**Best for simplicity and predictable billing: Option 1, staying on a competitive flat tariff.** It is sensible if ToU quotes are unattractive or if half-hourly data show too much unavoidable peak usage.

**Best for a homeowner who strongly values modest backup and finds an exceptional deal: Option 3, a roughly 5 kWh battery.** Financially it is weak at normal installed prices, but it is the only battery size that could become reasonable if installed very cheaply and paired with a reliably wide tariff spread.

**Best avoided for battery-only arbitrage: Option 4, a roughly 10 kWh battery.** It costs substantially more but is unlikely to generate materially more savings because the household may not have enough high-price demand to use its additional capacity consistently.

---

## Conclusion

For this English household as at 14 July 2026, **a home battery without solar panels is unlikely to be financially worthwhile on electricity-price arbitrage alone**.

A roughly 5 kWh battery may save around **£205–£210 per year** beyond a ToU tariff, but a likely installed cost around **£4,050** implies a simple payback near **20 years** and a ten-year arbitrage return well below the capital cost. A roughly 10 kWh battery is less attractive still: it is more expensive, more likely to be under-used and may have a simple payback near **30 years**.

By contrast, changing to a suitable time-of-use tariff without a battery could save around **£330 annually**, or roughly **£3,300 over ten years**, without equipment cost. That is the best expected financial outcome.

The strongest argument against this conclusion is that future smart tariffs could offer wider and more persistent overnight-to-peak price spreads, while the household’s substantial electricity demand may provide dependable discharge opportunities. But tariff structures, eligibility rules and price spreads are too uncertain to treat as bankable for a ten-year battery investment.

The practical recommendation is clear: **analyse half-hourly consumption data, obtain tariff quotes, switch tariff first, and buy a battery only if the data and quotations show an unusually favourable case.**

### Sources

- [S1] [Octopus Energy tariffs July 2026 | Fixed, Tracker, Agile, Go](https://www.energyplus.co.uk/suppliers/octopus-energy/tariffs)
- [S2] [Compare energy – Gas and Electricity Tariffs](https://www.britishgas.co.uk/energy.html)
- [S3] [How do I get the energy price cut from April 2026? | Octopus Energy](https://octopus.energy/blog/april-2026-price-cap-updates/)
- [S4] [What are the best SEG rates? | All 38 tariffs ranked [2026]](https://www.sunsave.energy/solar-panels-advice/exporting-to-the-grid/best-seg-rates)
- [S5] [EDF's EV Tariffs For Your Car And Home | EV Tariffs | EDF](https://www.edfenergy.com/electric-cars/ev-tariffs)
- [S6] [Clean flexibility roadmap (accessible webpage) - GOV.UK](https://www.gov.uk/government/publications/clean-flexibility-roadmap/clean-flexibility-roadmap)
- [S7] [Best EV energy tariffs that offer the cheapest EV charging](https://www.smarthomecharge.co.uk/features/best-ev-energy-tariffs/)
- [S8] [Octopus Flux | Energy Tariff Designed for Solar & Batteries | Octopus Energy](https://octopus.energy/smart/flux/)
- [S9] [The best solar batteries of 2026 | Researched and reviewed](https://www.theecoexperts.co.uk/solar-panels/the-best-storage-batteries)
- [S10] [GivEnergy Battery Review 2026 | Is It Worth It?](https://www.greentechrenewables.co.uk/battery-storage/givenergy-review)
- [S11] [Best Energy Deals & Quotes - Cheapest Gas & Electric 2026 | EDF](https://www.edfenergy.com/gas-and-electricity)
- [S12] [Our variable tariff prices](https://www.eonnext.com/electricity-and-gas/tariff-prices)
- [S13] [Choosing the Best Energy Tariffs - How to pick the right one for you | EDF](https://www.edfenergy.com/energywise/choosing-best-energy-tariff)
- [S14] [April 2026 energy price cap to fall by 7% | EDF](https://www.edfenergy.com/energywise/energy-price-cap-april-2026)
- [S15] [Energy Price Cap: Everything You Need To Know | OVO Energy](https://www.ovoenergy.com/pricecap)
- [S16] [UK electricity & gas fixed tariffs | Find the right tariff for you](https://www.eonnext.com/tariffs)
- [S17] [Energy prices from April 2026, and what they mean for you | Octopus Energy](https://octopus.energy/blog/energy-prices-from-july-2026-and-what-they-mean-for-you/)
- [S18] [Compare EV tariffs: Enjoy lower night prices](https://www.eonnext.com/tariffs/next-drive)
- [S19] [Ofgem's energy price cap explained | Energy price is falling.](https://www.eonnext.com/electricity-and-gas/price-cap)
- [S20] [EDF to give two extra off-peak hours on all EV tariffs from April | EDF](https://www.edfenergy.com/media-centre/edf-give-two-extra-peak-hours-all-ev-tariffs-april)
- [S21] [Smart Tariffs - Terms and Conditions | Octopus Energy](https://octopus.energy/policies/smart-tariffs-terms-and-condition/)
- [S22] [EDF launches cheapest overnight charging on EV tariffs | EDF](https://www.edfenergy.com/media-centre/edf-launches-cheapest-overnight-charging-ev-tariffs)
- [S23] [PRODUCT BROCHURE 2025 - GivEnergy](https://givenergy.co.uk/wp-content/uploads/GivEnergy-UK-Product-Brochure.pdf)
- [S24] [Battery-Fox ESS](https://www.fox-ess.com/products/battery)
- [S25] [GivEnergy | Home Battery Storage for UK Homes](https://givenergy.co.uk/what-is-round-trip-efficiency-in-battery-storage/)
- [S26] [Smart Export Guarantee & Tariff | Solar Energy | EDF](https://www.edfenergy.com/energy-efficiency/smart-export-tariff)
- [S27] [Solar Tariffs - Compare and Save on Energy Bills | EDF](https://www.edfenergy.com/solar/solar-tariffs)
- [S28] [Understanding time of use tariffs and energy bills.](https://www.eonnext.com/energy/guides/time-of-use-tariffs)
- [S29] [UK Energy Blog: News, Advice & Guides | E.ON Next](https://www.eonnext.com/blog)
- [S30] [EDF launches first-of-its-kind three-rate tariff with free electricity hours | EDF](https://www.edfenergy.com/media-centre/edf-launches-first-its-kind-three-rate-tariff-free-electricity-hours)
- [S31] [Smart meter tariffs by E.ON Next | Use electricity when it is cheaper](https://www.eonnext.com/tariffs/smart-tariffs)
- [S32] [Energy prices are rising, should I fix my tariff?](https://www.eonnext.com/blog/energy-prices-are-rising-should-i-fix-my-tariff)
- [S33] [Smart Export Guarantee (SEG) Tariffs | EDF Small Business](https://www.edfenergy.com/energywise/small-business-SEG-tariffs)

[S1]: https://www.energyplus.co.uk/suppliers/octopus-energy/tariffs
[S2]: https://www.britishgas.co.uk/energy.html
[S3]: https://octopus.energy/blog/april-2026-price-cap-updates/
[S4]: https://www.sunsave.energy/solar-panels-advice/exporting-to-the-grid/best-seg-rates
[S5]: https://www.edfenergy.com/electric-cars/ev-tariffs
[S6]: https://www.gov.uk/government/publications/clean-flexibility-roadmap/clean-flexibility-roadmap
[S7]: https://www.smarthomecharge.co.uk/features/best-ev-energy-tariffs/
[S8]: https://octopus.energy/smart/flux/
[S9]: https://www.theecoexperts.co.uk/solar-panels/the-best-storage-batteries
[S10]: https://www.greentechrenewables.co.uk/battery-storage/givenergy-review
[S11]: https://www.edfenergy.com/gas-and-electricity
[S12]: https://www.eonnext.com/electricity-and-gas/tariff-prices
[S13]: https://www.edfenergy.com/energywise/choosing-best-energy-tariff
[S14]: https://www.edfenergy.com/energywise/energy-price-cap-april-2026
[S15]: https://www.ovoenergy.com/pricecap
[S16]: https://www.eonnext.com/tariffs
[S17]: https://octopus.energy/blog/energy-prices-from-july-2026-and-what-they-mean-for-you/
[S18]: https://www.eonnext.com/tariffs/next-drive
[S19]: https://www.eonnext.com/electricity-and-gas/price-cap
[S20]: https://www.edfenergy.com/media-centre/edf-give-two-extra-peak-hours-all-ev-tariffs-april
[S21]: https://octopus.energy/policies/smart-tariffs-terms-and-condition/
[S22]: https://www.edfenergy.com/media-centre/edf-launches-cheapest-overnight-charging-ev-tariffs
[S23]: https://givenergy.co.uk/wp-content/uploads/GivEnergy-UK-Product-Brochure.pdf
[S24]: https://www.fox-ess.com/products/battery
[S25]: https://givenergy.co.uk/what-is-round-trip-efficiency-in-battery-storage/
[S26]: https://www.edfenergy.com/energy-efficiency/smart-export-tariff
[S27]: https://www.edfenergy.com/solar/solar-tariffs
[S28]: https://www.eonnext.com/energy/guides/time-of-use-tariffs
[S29]: https://www.eonnext.com/blog
[S30]: https://www.edfenergy.com/media-centre/edf-launches-first-its-kind-three-rate-tariff-free-electricity-hours
[S31]: https://www.eonnext.com/tariffs/smart-tariffs
[S32]: https://www.eonnext.com/blog/energy-prices-are-rising-should-i-fix-my-tariff
[S33]: https://www.edfenergy.com/energywise/small-business-SEG-tariffs

### Analyzed URLs

1. [Octopus Energy tariffs July 2026 | Fixed, Tracker, Agile, Go](https://www.energyplus.co.uk/suppliers/octopus-energy/tariffs)
2. [Compare energy – Gas and Electricity Tariffs - British Gas](https://www.britishgas.co.uk/energy.html)
3. [How do I get the energy price cut from April 2026?](https://octopus.energy/blog/april-2026-price-cap-updates/)
4. [EDF, E.on and Octopus Energy launch limited trials of 'low Standing ...](https://www.facebook.com/MoneySavingExpert/posts/edf-eon-and-octopus-energy-launch-limited-trials-of-low-standing-charge-tariffsh/1504915801672684/)
5. [What are the best SEG rates? | All 38 tariffs ranked [2026] - Sunsave](https://www.sunsave.energy/solar-panels-advice/exporting-to-the-grid/best-seg-rates)
6. [Electric vehicle tariffs for cheaper home charging - EDF Energy](https://www.edfenergy.com/electric-cars/ev-tariffs)
7. [Octopus Energy - Facebook](https://www.facebook.com/octopusenergy/posts/energy-prices-are-dropping-from-april-1st-the-government-is-cutting-some-levies-/1237731181908639/)
8. [Clean flexibility roadmap (accessible webpage) - GOV.UK](https://www.gov.uk/government/publications/clean-flexibility-roadmap/clean-flexibility-roadmap)
9. [Best EV energy tariffs that offer the cheapest EV charging](https://www.smarthomecharge.co.uk/features/best-ev-energy-tariffs/)
10. [Octopus Flux | Energy Tariff Designed for Solar & Batteries](https://octopus.energy/smart/flux/)
11. [The best solar batteries of 2026 | Researched and reviewed](https://www.theecoexperts.co.uk/solar-panels/the-best-storage-batteries)
12. [GivEnergy Battery Review 2026 | Is It Worth It?](https://www.greentechrenewables.co.uk/battery-storage/givenergy-review)
13. [Compare our best energy deals on gas and electricity tariffs](https://www.edfenergy.com/gas-and-electricity)
14. [Change or Renew Your Energy Tariff | New & Existing Customers](https://www.edfenergy.com/gas-and-electricity/change-energy-tariff)
15. [Our variable tariff prices - E.ON Next](https://www.eonnext.com/electricity-and-gas/tariff-prices)
16. [Choosing the Best Energy Tariffs - How to pick the right one for you](https://www.edfenergy.com/energywise/choosing-best-energy-tariff)
17. [April 2026 energy price cap to fall by 7% | EDF](https://www.edfenergy.com/energywise/energy-price-cap-april-2026)
18. [Energy Price Cap: Everything You Need To Know](https://www.ovoenergy.com/pricecap)
19. [UK electricity & gas fixed tariffs | Find the right tariff for you - E.ON Next](https://www.eonnext.com/tariffs)
20. [Energy prices from July 2026, and what they mean for you](https://octopus.energy/blog/energy-prices-from-july-2026-and-what-they-mean-for-you/)
21. [Compare EV tariffs: Enjoy lower night prices - E.ON Next](https://www.eonnext.com/tariffs/next-drive)
22. [Ofgem's energy price cap is rising. - E.ON Next](https://www.eonnext.com/electricity-and-gas/price-cap)
23. [EDF to give two extra off-peak hours on all EV tariffs from April](https://www.edfenergy.com/media-centre/edf-give-two-extra-peak-hours-all-ev-tariffs-april)
24. [Smart Tariffs - Terms and Conditions - Octopus Energy](https://octopus.energy/policies/smart-tariffs-terms-and-condition/)
25. [EDF launches cheapest overnight charging on EV tariffs](https://www.edfenergy.com/media-centre/edf-launches-cheapest-overnight-charging-ev-tariffs)
26. [PRODUCT BROCHURE 2025 - GivEnergy](https://givenergy.co.uk/wp-content/uploads/GivEnergy-UK-Product-Brochure.pdf)
27. [Powerwall – Home Battery Storage | Tesla United Kingdom](https://www.tesla.com/en_gb/powerwall)
28. [Tesla Powerwall 2 Datasheet](https://www.tesla.com/sites/default/files/pdfs/powerwall/Powerwall%202_AC_Datasheet_en_AU.pdf)
29. [Battery - Fox ESS](https://www.fox-ess.com/products/battery)
30. [Home Battery Storage for UK Homes - GivEnergy](https://givenergy.co.uk/what-is-round-trip-efficiency-in-battery-storage/)
31. [Smart Export Guarantee & Tariff | Solar Energy | EDF](https://www.edfenergy.com/energy-efficiency/smart-export-tariff)
32. [Solar Tariffs - Compare and Save on Energy Bills | EDF](https://www.edfenergy.com/solar/solar-tariffs)
33. [Understanding time of use tariffs and energy bills. - E.ON Next](https://www.eonnext.com/energy/guides/time-of-use-tariffs)
34. [UK Energy Blog: News, Advice & Guides | E.ON Next](https://www.eonnext.com/blog)
35. [EDF launches first-of-its-kind three-rate tariff with free electricity hours](https://www.edfenergy.com/media-centre/edf-launches-first-its-kind-three-rate-tariff-free-electricity-hours)
36. [Smart meter tariffs by E.ON Next | Use electricity when it is cheaper](https://www.eonnext.com/tariffs/smart-tariffs)
37. [Energy prices are rising, should I fix my tariff? - E.ON Next](https://www.eonnext.com/blog/energy-prices-are-rising-should-i-fix-my-tariff)
38. [Smart Export Guarantee (SEG) Tariffs | EDF Small Business](https://www.edfenergy.com/energywise/small-business-SEG-tariffs)

<details>
<summary><strong>Raw collected findings (33 sources)</strong></summary>

**1. [S1] [Octopus Energy tariffs July 2026 | Fixed, Tracker, Agile, Go](https://www.energyplus.co.uk/suppliers/octopus-energy/tariffs)**

The useful takeaway is that a battery may create value only where a genuinely large, reliable spread exists between cheap overnight imports and the household’s expensive peak-period imports. Agile is potentially relevant because it has half-hourly pricing and can reward battery discharge during the cited 16:00–19:00 peak, but it is volatile and its claimed averages are not personalised. Go/Intelligent Go show potentially attractive overnight rates, but the page presents them as EV tariffs and says Intelligent Go requires a compatible EV or charger; eligibility must be confirmed and should not be assumed for a non-EV household. Flux is described primarily as a solar-plus-battery/export product, so it is not clear from this page that it is suitable or financially advantageous for a battery-only household with no solar export. A tariff change without a battery could still provide meaningful savings if the home office demand or other usage can naturally fall in cheap ToU periods, but the stated high daytime office load may limit this. For a robust comparison, obtain official postcode-specific tariff labels and eligibility terms, half-hourly smart-meter data covering at least 12 months, and battery quotations specifying installed cost, usable kWh, continuous and peak power, round-trip efficiency, warranty throughput/retained-capacity terms, and replacement assumptions. Independently verify every July 2026 tariff claim with Octopus and Ofgem primary materials before calculating annual costs or payback.

**2. [S2] [Compare energy – Gas and Electricity Tariffs](https://www.britishgas.co.uk/energy.html)**

For a household with a smart meter and substantial daytime home-office use, Charge Power creates a plausible battery-arbitrage mechanism: buy electricity between midnight and 5am at 50% of the tariff’s ordinary unit rate, store part of it, and use it in daytime hours. However, the discount applies to all metered electricity in that window, including electricity consumed directly overnight; it is not exclusively electricity stored in a battery. Therefore a financial model must separate direct overnight load from battery charging, apply battery round-trip losses, and limit battery throughput to the household’s actual daytime demand and battery power/capacity. The source confirms a five-hour off-peak window and variable tariff pricing, but it does not state Charge Power’s undiscounted unit rate, standing charge, peak rate, export terms, or battery-installation costs. Those rates must be obtained as a postcode-specific quote and compared with competitive flat tariffs and other time-of-use tariffs before calculating annual savings or payback. PeakSave may offer modest battery-free savings for flexible Sunday-afternoon usage, but it is only a limited weekly window and is unlikely on its own to materially change the economics of a high-usage household.

**3. [S3] [How do I get the energy price cut from April 2026? | Octopus Energy](https://octopus.energy/blog/april-2026-price-cap-updates/)**

For a 14 July 2026 assessment, this source supports adjusting both flat-rate and smart-tariff assumptions for the April 2026 reduction in electricity policy costs, rather than assuming older unit rates persist. The stated reduction is principally in unit rates, so a high-use household may see a larger cash effect than the supplier’s ‘typical household’ illustration, but exact savings require the household’s regional tariff rates and standing charge. The source also indicates that the green-levy component is intended to remain reduced until 2029, while wholesale, network and other costs remain uncertain. This reinforces that tariff-spread and future-price sensitivity analysis is essential for a battery case. It does not itself establish whether a 5 kWh or 10 kWh battery pays back.

**4. [S4] [What are the best SEG rates? | All 38 tariffs ranked [2026]](https://www.sunsave.energy/solar-panels-advice/exporting-to-the-grid/best-seg-rates)**

For a non-solar UK household, the useful evidence is the possible import-price spread: EDF Empower Fixed is quoted at 10.361p/kWh from 1am–4am, 20.22p/kWh at most other times, and 30.22p/kWh from 4pm–7pm; 100green Tide Smart is quoted at about 7.43p/kWh overnight, 34.63p/kWh normally, and 46.96p/kWh at 4pm–7pm. A battery could capture some overnight-to-day/peak spread, subject to usable capacity, charge/discharge power, losses, and the amount of demand occurring after charging. The cited 100green tariff is particularly risky without sufficient battery capacity or load shifting because its daytime rates are high. A tariff-only switch may already save money if enough load can move overnight, but the household’s daytime home-office usage limits manual shifting. The webpage provides no installed battery prices, usable capacities, battery efficiency, degradation, warranty, replacement cost, taxes, or reliable non-solar annual-cost analysis; these must be obtained from primary supplier tariffs, installer quotations, and manufacturer documentation. Its solar/SEG savings results should be excluded from the requested financial model.

**5. [S5] [EDF's EV Tariffs For Your Car And Home | EV Tariffs | EDF](https://www.edfenergy.com/electric-cars/ev-tariffs)**

As at the supplier’s cited April 2026 comparison date, EDF advertises overnight electricity at 6.49–6.99p/kWh for seven hours, 23:00–06:00, on fixed EV tariffs requiring a smart meter. This is potentially attractive for a battery that imports cheap overnight electricity and discharges during expensive daytime/early-evening periods. The household’s stated smart meter satisfies one stated requirement, but the page presents these as EV tariffs and does not establish whether a non-EV, battery-only household qualifies. It also omits the corresponding normal/peak rate and standing charge, which are essential: battery arbitrage savings equal avoided peak imports less off-peak charging imports after round-trip losses. The seven-hour window is normally ample to charge a 5–10 kWh battery if its inverter/charger permits roughly 2–3 kW or more, but actual usable charging is constrained by overnight household load, battery charge rate, and the fact that direct overnight loads should not be counted as battery-charged energy. EDF’s note that rates and off-peak hours vary by supplier, plus fixed-term exit fees, supports treating this tariff as an illustrative current low-rate benchmark rather than assuming the spread is guaranteed for ten years.

**6. [S6] [Clean flexibility roadmap (accessible webpage) - GOV.UK](https://www.gov.uk/government/publications/clean-flexibility-roadmap/clean-flexibility-roadmap)**

The page provides strong policy-level support for considering a time-of-use tariff before purchasing a battery: savings arise only where demand can actually be moved to cheaper periods. It is particularly pertinent that Ofgem acknowledges households with inflexible, time-specific consumption may not benefit as much. For the stated daytime home-office load, a battery could potentially shift some cheap overnight electricity into daytime/peak use, but this source does not quantify tariff spreads, battery efficiency, installed cost, warranty, degradation, or household savings. It therefore supports the qualitative case for smart-tariff flexibility and flags tariff/market uncertainty, but cannot by itself establish whether either a 5 kWh or 10 kWh battery pays back.

**7. [S7] [Best EV energy tariffs that offer the cheapest EV charging](https://www.smarthomecharge.co.uk/features/best-ev-energy-tariffs/)**

The source suggests overnight time-of-use rates of roughly 7.5–8.95p/kWh for five to six hours, versus the household’s stated 24.7p/kWh flat rate. This establishes a potentially valuable tariff-arbitrage opportunity even without a battery if some demand can be moved overnight. For a battery, each delivered kWh could be materially cheaper than daytime electricity after round-trip losses, but the webpage provides no corresponding daytime/peak unit rates, battery compatibility or tariff eligibility for a non-EV household, installed battery prices, degradation data, warranties, or replacement costs. It therefore supports investigating a suitable off-peak tariff, but cannot by itself establish whether a 5 kWh or 10 kWh battery pays back. Its warning that dynamic Agile prices can spike also supports treating tariff spreads as uncertain in sensitivity analysis.

**8. [S8] [Octopus Flux | Energy Tariff Designed for Solar & Batteries | Octopus Energy](https://octopus.energy/smart/flux/)**

Flux cannot be used for the stated no-solar household: Octopus explicitly requires a solar system, a battery, half-hourly smart-meter data, and both Octopus import and export tariffs. The page nonetheless shows the relevant battery-arbitrage model: charge during a three-hour 02:00–05:00 low-price window and discharge during the 16:00–19:00 peak window. This may be achievable through a different battery-compatible import tariff, but Flux itself must be excluded from the financial comparison. The page also warns that rates are flexible and battery scheduling may be manual, so any financial model should not treat advertised time bands or price spreads as guaranteed for ten years.

**9. [S9] [The best solar batteries of 2026 | Researched and reviewed](https://www.theecoexperts.co.uk/solar-panels/the-best-storage-batteries)**

The page supports using approximately 4.5–4.8 kWh as usable capacity for a nominal 5.12 kWh Dura5, with a quoted installed cost around £4,000, a maximum discharge rate of 5.63 kW, and a 10-year warranty. It also indicates that battery-system costs can readily exceed £5,000 and that inverters and backup hardware may add material cost. For a battery-only, tariff-arbitrage assessment, however, the source’s claimed savings are not directly transferable: the stated “up to 90%” saving explicitly depends on solar plus smart tariffs. The page contains no tariff spread, actual round-trip efficiency, cycling profile, degradation model, or battery-only financial calculation. Its pricing should therefore be treated as a preliminary, marketing-adjacent benchmark and checked against several itemised installed quotations and manufacturer documentation.

**10. [S10] [GivEnergy Battery Review 2026 | Is It Worth It?](https://www.greentechrenewables.co.uk/battery-storage/givenergy-review)**

For an indicative no-solar assessment, this page supports modelling a 5.2 kWh unit as 4.68 kWh usable, 2.6 kW maximum output, roughly 92–95% round-trip efficient, costing about £3,800–£4,300 installed; and a 9.5 kWh unit as 8.55 kWh usable, 3.6 kW output, costing about £5,800–£6,500. The 2.6 kW/3.6 kW power ceilings matter because concurrent office and household demand above those levels would still be bought at the tariff’s peak price. The roughly 10 W standby draw and possible up-to-20% ten-year degradation should be included in long-term calculations, while the warranty indicates capacity protection only below 70% and does not establish replacement-cost coverage after ten years. The claimed tariff integrations suggest automated overnight charging is technically possible, reducing the need for manual load shifting. Nevertheless, the page provides no current off-peak/peak prices or evidence that a battery will cycle fully on this household’s actual demand pattern; direct overnight load may reduce energy available to charge it, and daytime office demand may not coincide with peak-price periods. Its quoted £300–£400 annual savings are explicitly “with solar” and should not be used as evidence of savings without solar. The homeowner should obtain half-hourly smart-meter data, supplier-specific tariff terms and price spreads, confirmation of import/export and battery-control functionality, and multiple fixed-price quotations including inverter, backup capability, warranty labour, and post-warranty replacement costs.

**11. [S11] [Best Energy Deals & Quotes - Cheapest Gas & Electric 2026 | EDF](https://www.edfenergy.com/gas-and-electricity)**

EDF evidence supports testing a time-of-use strategy: a smart meter can access tariffs with a seven-hour 11pm–6am low-price window, and EDF cites off-peak EV rates of 6.49–6.99p/kWh as of 24 April 2026. A household with high daytime home-office demand may benefit from a suitable time-of-use tariff even without a battery if it can move some demand overnight; a battery could additionally shift cheap overnight imports into expensive daytime/peak periods. Financial viability nevertheless depends on the actual all-in peak and off-peak unit rates, standing charges, tariff eligibility, and the battery’s usable delivered energy and installed cost. EDF’s solar-and-battery £0-bill example is not relevant evidence for a battery-only decision because it assumes 2,500 kWh/year of solar generation. The homeowner should obtain a personalised EDF quote confirming whether its EV or other time-of-use tariffs permit battery-only participation, the full daytime/peak rates and standing charge, and compare those against other suppliers’ tariffs before modelling battery payback.

**12. [S12] [Our variable tariff prices](https://www.eonnext.com/electricity-and-gas/tariff-prices)**

For this household, the page supports caution about conventional Economy 7/Economy 10 without a battery: a home office using substantial electricity in daytime working hours is likely to consume much of its power outside the cheap overnight window, while only a small amount can be manually shifted. Therefore, a dual-rate tariff without storage could be worse than a competitive flat tariff unless interval data show enough overnight/off-peak consumption. A battery could in principle charge in the stated off-peak window and supply daytime demand, making a TOU tariff more usable; however, this source supplies no actual unit-rate spread, battery installed price, usable capacity, efficiency, power rating, degradation, or warranty data. Its tariff download for the household’s postcode and meter arrangement is needed before calculating annual savings or battery payback.

**13. [S13] [Choosing the Best Energy Tariffs - How to pick the right one for you | EDF](https://www.edfenergy.com/energywise/choosing-best-energy-tariff)**

For this English household, the smart meter means a TOU tariff can be considered even without a battery. Given the high daytime home-office load and limited ability to shift demand manually, tariff savings without a battery may be limited unless the selected tariff has low daytime rates or predictable off-peak windows that match unavoidable use. A battery could charge in cheaper off-peak periods and discharge during higher-priced periods, but this webpage supplies no tariff rates or battery economics needed to calculate savings or payback. EDF’s claimed free-electricity events should not be included as dependable battery revenue: the supplier expressly says they are not guaranteed and depend on location and grid surplus. The key useful conclusion from this source is that interval consumption data from the smart meter should be analysed before selecting either a TOU tariff or battery size.

**14. [S14] [April 2026 energy price cap to fall by 7% | EDF](https://www.edfenergy.com/energywise/energy-price-cap-april-2026)**

For the requested 14 July 2026 assessment, this page supplies a useful but time-limited benchmark: the April–June 2026 average default electricity rate was 24.67p/kWh including 5% VAT, plus 57.21p/day. At 9,000 kWh/year, electricity consumption alone at that unit rate would be about £2,220/year, before roughly £209/year of electricity standing charges, or approximately £2,429/year in total if these rates persisted for a year. The rate is close to the stated 24.7p/kWh flat-rate assumption. However, the page explicitly says the cap changes quarterly; therefore it should not be treated as a fixed full-year 2026/27 price. Its claim that trackers may save money and that smart meters enable access to better offers supports investigating tariff switching, but it provides no evidence to quantify time-of-use savings or battery arbitrage. Separate primary evidence is required for actual half-hourly tariff rates, battery installed costs, usable capacity, charge/discharge power, efficiency, degradation, warranty terms, and tax/regulatory treatment.

**15. [S15] [Energy Price Cap: Everything You Need To Know | OVO Energy](https://www.ovoenergy.com/pricecap)**

As of 1 July–30 September 2026, this OVO page says the Ofgem price cap is £1,663 for a ‘typical’ direct-debit household, but stresses that this is not a cap on a household’s total bill: unit rates, standing charges, and actual consumption determine cost. Its cited typical electricity use is only 2,500 kWh/year, far below the proposed household’s 9,000 kWh/year, so the headline annual figure should not be used to estimate that household’s bill. The page also supports the general proposition that smart-meter customers may obtain rewards for off-peak use, but provides neither a defined tariff nor rates. It therefore cannot substantiate the financial case for a 5 kWh or 10 kWh battery; primary tariff quotes, half-hourly consumption data, and battery technical/installed-price evidence are still required.

**16. [S16] [UK electricity & gas fixed tariffs | Find the right tariff for you](https://www.eonnext.com/tariffs)**

The strongest usable evidence is that E.ON Next offered smart-meter time-of-use products in mid-2026, including Smart Saver with low-priced periods spanning 5am–4pm and 7pm–2am, plus a 2am–5am super-off-peak window. This is potentially unusually suitable for the stated home-office demand: substantial daytime consumption may already fall in an off-peak period, allowing meaningful tariff-only savings without a battery. The advertised Pumped structure explicitly treats 4pm–7pm as peak and 10pm–6am as super-off-peak, which is the relevant battery-arbitrage window. However, this household uses gas heating and may not qualify for the heat-pump-specific tariff; Drive Smart requires a compatible EV, so neither should be assumed available. The 8p/kWh Drive Smart price is not applicable unless the household has the qualifying EV. Supplier savings figures are based on a lower-use, assumed load profile and regional rates, so they cannot establish savings for 9,000 kWh/year. For the full financial model, obtain postcode-specific unit rates and standing charges, tariff terms and eligibility in writing, and half-hourly smart-meter data. The page offers no evidence that either a 5 kWh or 10 kWh battery pays back; it only supports assessing tariff switching first and treating a possible £200 supplier battery discount as a modest reduction in installed cost.

**17. [S17] [Energy prices from April 2026, and what they mean for you | Octopus Energy](https://octopus.energy/blog/energy-prices-from-july-2026-and-what-they-mean-for-you/)**

As at 14 July 2026, the page reports an average direct-debit variable-tariff electricity cap of 26.11p/kWh plus a 57.19p/day standing charge, including VAT. The household’s stated 24.7p/kWh flat rate is therefore already about 1.41p/kWh below this national average capped-variable unit-rate benchmark, before considering standing charges. At 9,000 kWh/year, that unit-rate difference is about £127/year, although a valid comparison also requires the actual regional standing charge and tariff terms. The source supports using region-specific rates rather than the national average and confirms that high-use households have bills materially different from the headline ‘typical bill.’ It also indicates that Economy 7/time-of-use arrangements can have different capped rates, but supplies no actual off-peak/peak price spread. Therefore, it supports the baseline-tariff part of the assessment only; tariff-specific quotations and battery evidence are still required to model battery charging arbitrage, annual cycles, losses, payback, and 10-year outcomes.

**18. [S18] [Compare EV tariffs: Enjoy lower night prices](https://www.eonnext.com/tariffs/next-drive)**

For tariff modelling, this source supports an illustrative 00:00–06:00 overnight import price of 9p/kWh on Next Drive, compared with a cited national-average standard-variable rate of 24.674p/kWh: a gross spread of about 15.7p/kWh before battery losses. With a battery at 90% round-trip efficiency, each delivered kWh charged at 9p costs roughly 10p, implying an indicative avoided-cost margin of about 14.7p/kWh if it genuinely displaces electricity otherwise bought at 24.674p/kWh. The Smart version advertises 8p/kWh off-peak but requires both an eligible EV and compatible charger, so it is not a valid assumption for the no-solar/no-EV household described. The regular Next Drive page also frames eligibility around EV use, therefore the homeowner should obtain written confirmation that a battery-only household can join. The six-hour overnight window is potentially adequate for a 5 kWh battery and often a 10 kWh battery, subject to inverter charge rate and direct overnight household consumption. Because the home office load is mainly daytime, batteries may have usable discharge demand, but the source does not supply the daytime/peak rate of Next Drive, installed battery costs, battery efficiency, warranty/degradation, or battery eligibility. It therefore supports only one input to the financial assessment, not a conclusion that either battery will pay back.

**19. [S19] [Ofgem's energy price cap explained | Energy price is falling.](https://www.eonnext.com/electricity-and-gas/price-cap)**

For a 9,000 kWh/year household, the source supports the general conclusion that electricity expenditure remains strongly usage-dependent: a £0.247/kWh flat rate implies approximately £2,223/year for electricity alone, before standing charge. The page also confirms that a smart-meter household may access a time-of-use tariff with off-peak, super-off-peak and peak periods, which is the essential prerequisite for battery arbitrage. However, it supplies no actual tariff rates or time windows, and no battery installation, efficiency, degradation, warranty, replacement or export information. Therefore it cannot substantiate claims that a battery will pay back. Its strongest contribution is that switching tariff may offer savings without capital expenditure, while battery economics will depend on the real off-peak-to-peak price spread, the home’s half-hourly load profile, and whether enough daytime/peak demand exists to use stored energy after accounting for charging losses.

**20. [S20] [EDF to give two extra off-peak hours on all EV tariffs from April | EDF](https://www.edfenergy.com/media-centre/edf-give-two-extra-peak-hours-all-ev-tariffs-april)**

For the specified English household, EDF’s seven-hour 23:00–06:00 TOU window could technically accommodate overnight charging of either a roughly 5 kWh or 10 kWh battery, subject to the battery/inverter charge-power limit and simultaneous overnight household load. The page also confirms smart-meter and half-hourly-reading requirements, which the household can meet. It supports analysing tariff switching alone—some ordinary overnight demand can be moved onto the low rate without a battery—and battery arbitrage, where stored overnight electricity is used during expensive daytime/peak periods. The household’s high daytime home-office load makes it more plausible that stored energy can be discharged usefully, but the article supplies no prices or proof that a non-EV household can join these EV-labelled tariffs. Therefore it cannot establish financial worth: obtain EDF’s actual quoted off-peak and other-period rates, standing charge, tariff terms and explicit eligibility before using this tariff in a payback model.

**21. [S21] [Smart Tariffs - Terms and Conditions | Octopus Energy](https://octopus.energy/policies/smart-tariffs-terms-and-condition/)**

For the proposed England household, this source supports considering a smart tariff before buying a battery: half-hourly pricing may reward moving flexible demand away from expensive periods, though the household’s largely unavoidable daytime home-office demand limits manual load shifting. A battery could charge in lower-price periods and discharge during the 16:00–19:00 Intelligent Flux period, but this page provides no tariff rates, battery-cycle revenue figures, or evidence that a full daily cycle is achievable. Octopus Go and Intelligent Octopus Go should not be assumed available merely because the home has a battery: both are stated to require a qualifying EV/PHEV. Intelligent Octopus Flux is the most directly relevant named tariff, but it requires an Octopus-approved home battery and has a three-hour 16:00–19:00 flux period. Dynamic Agile pricing can be an alternative, but Octopus explicitly warns that historical prices do not predict future costs and prices can rise; financial modelling should therefore use conservative tariff-spread scenarios rather than advertised or historic best-case savings. Smart-meter compatibility and reliable half-hourly readings are also necessary, and the supplier warns that smart tariffs and associated technology may be unreliable or changed over time.

**22. [S22] [EDF launches cheapest overnight charging on EV tariffs | EDF](https://www.edfenergy.com/media-centre/edf-launches-cheapest-overnight-charging-ev-tariffs)**

As of 1 April 2026, EDF advertised 7-hour nightly off-peak windows (23:00–06:00) at 6.99p/kWh on GoElectric and 6.49p/kWh on Pod Point Plug & Power. The household’s smart meter is consistent with the stated technical requirement, subject to half-hourly readings and tariff eligibility. This creates potentially large savings even without a battery if meaningful demand can be moved overnight, but the described household has predominantly daytime home-office demand and limited manual load shifting. A battery could shift some 6.49–6.99p/kWh overnight energy to daytime use, but the page provides neither the daytime rate nor battery costs, efficiency, capacity, power limits, degradation, or warranty terms. It therefore cannot establish that a 5 kWh or 10 kWh battery pays back. The page’s illustrative EDF load split—about 42% off-peak (1,950 of 4,650 kWh)—is an EV-oriented example and should not be assumed for this household; actual half-hourly smart-meter data is essential. Before relying on this tariff for battery arbitrage, obtain EDF’s full unit-rate quote including daytime/peak rate, standing charge, exit fees, whether a non-EV household may take the tariff, and whether battery charging/discharging is permitted or affects any smart-charging terms.

**23. [S23] [PRODUCT BROCHURE 2025 - GivEnergy](https://givenergy.co.uk/wp-content/uploads/GivEnergy-UK-Product-Brochure.pdf)**

No relevant substantive information can be extracted from this source as provided. A text-accessible version of the PDF, OCR output, or the original webpage URL/content would be needed before it can be assessed as evidence for whether a battery without solar panels is financially worthwhile in England.

**24. [S24] [Battery-Fox ESS](https://www.fox-ess.com/products/battery)**

This source confirms that Battery-Fox offers modular high-voltage storage systems whose nominal capacities could cover roughly 5 kWh and 10 kWh configurations, particularly the EP5 and EP11 families. However, nominal capacity is not usable capacity, and the page supplies none of the technical or commercial evidence required to estimate annual arbitrage savings, daily cycle feasibility, payback, or a 10-year financial result in England. Its claims of “High Efficiency,” “Safety Reliable,” and “Easy Installation” are unsupported marketing descriptors on this page. Product-specific datasheets, warranty documents, UK distributor/installer quotations, maximum charge/discharge power, round-trip efficiency, depth-of-discharge limits, throughput/cycle warranty, and compatible tariff-control arrangements would be needed before this product range could be included in the requested comparison.

**25. [S25] [GivEnergy | Home Battery Storage for UK Homes](https://givenergy.co.uk/what-is-round-trip-efficiency-in-battery-storage/)**

The page supports the technical strategy behind a battery-only installation: buy electricity overnight on an off-peak time-of-use tariff and use stored electricity when grid prices are higher. It claims a battery-only system averages £5,000 installed, £1,050 annual bill savings and a five-year ROI, but provides no battery capacity, usable capacity, power rating, round-trip efficiency, warranty/degradation assumptions, tariff rates, standing charges, cycling frequency, or methodology. Its claim should therefore be treated as an optimistic manufacturer benchmark, not an applicable calculation. For this household, high daytime office consumption may reduce the amount of demand available for discharge during expensive peak periods, while some cheap overnight electricity will be used directly rather than through the battery; both factors can materially reduce battery arbitrage savings. The page does not provide enough evidence to compare a 5 kWh versus 10 kWh battery or determine payback, though it does indicate that scalability and battery-only operation are possible.

**26. [S26] [Smart Export Guarantee & Tariff | Solar Energy | EDF](https://www.edfenergy.com/energy-efficiency/smart-export-tariff)**

For a UK household that will not install solar, this page supports excluding SEG/export income from the battery economics. A battery-only system should generally charge on a low-priced import period and discharge behind the meter during expensive periods; exporting grid-charged electricity cannot safely be treated as SEG-eligible income. The homeowner’s smart meter satisfies an important metering prerequisite for time-of-use tariffs, but EDF’s SEG offers, including their advertised 15p–18p/kWh rates, should not be used to justify the financial case unless the installer and supplier provide written confirmation of eligibility under the exact battery-only arrangement. Tariff and export rates are variable or time-limited, reinforcing the need for sensitivity analysis rather than reliance on promotional savings claims.

**27. [S27] [Solar Tariffs - Compare and Save on Energy Bills | EDF](https://www.edfenergy.com/solar/solar-tariffs)**

For a battery-only household, EDF’s Empower Fixed is potentially relevant: it explicitly accepts customers with “a battery,” requires a smart meter set to half-hourly readings, has no exit fee, and offers a 10p/kWh overnight charging discount. The page does not state the actual standard and overnight unit rates, charging window, battery installed price, battery VAT treatment, usable capacity, power limits, efficiency, or warranty; those must be obtained separately before calculating savings. EDF’s £0-bill and £800-plus-saving claims should not be used as evidence that a standalone battery pays back: the stated illustrative case includes 2,500 kWh/year solar generation, a 3 kWh battery, import/export tariffs, daytime solar self-consumption, off-peak grid top-ups, and peak discharge. SEG information is largely irrelevant absent solar generation, except that exported battery energy may be paid under an applicable export arrangement; its economics must be checked against the energy bought to charge it, losses, and tariff terms. The page confirms zero-rate VAT for solar energy-saving-material installations but does not establish that a standalone battery receives the same treatment. Overall, it supports considering a time-of-use battery tariff, but cannot by itself establish financial viability or provide the requested battery-only cost/payback calculation.

**28. [S28] [Understanding time of use tariffs and energy bills.](https://www.eonnext.com/energy/guides/time-of-use-tariffs)**

For the stated household, the page supports considering a TOU tariff because it already has a communicating smart meter and can provide half-hourly readings. The household’s substantial daytime home-office load may limit savings from manual load shifting, since the cited typical cheap period is overnight and peak periods include mornings and evenings. A battery could theoretically charge in the usual overnight off-peak window and discharge during costlier periods, but this page supplies no tariff prices or battery performance/cost data to establish whether that is financially worthwhile. The most useful action supported here is to obtain and analyse half-hourly smart-meter data: it can quantify how much use occurs overnight, during daytime work hours, and during morning/evening peak windows, which determines both tariff-only savings and the realistically usable daily battery throughput.

**29. [S29] [UK Energy Blog: News, Advice & Guides | E.ON Next](https://www.eonnext.com/blog)**

This source supports only the general proposition that smart tariffs and automated battery control may create savings by shifting charging and discharging in response to time-varying prices. The stated “average of £300/year” saving is a supplier marketing claim, appears to concern Next Optimise with solar/battery operation, and cannot be applied to a battery-only household without the underlying methodology, eligibility criteria, tariff rates, export assumptions, and installed system details. It does not establish whether a 5 kWh or 10 kWh battery pays back for the specified high-daytime-demand English household. Full linked articles and primary tariff, product, warranty, and installation-quote information would be required.

**30. [S30] [EDF launches first-of-its-kind three-rate tariff with free electricity hours | EDF](https://www.edfenergy.com/media-centre/edf-launches-first-its-kind-three-rate-tariff-free-electricity-hours)**

The page supports investigating tariff switching before buying a battery. A smart-meter household can access a three-period tariff with the expensive period limited to 4pm–7pm, while 6am–4pm and 7pm–11pm are amber and 11pm–6am is the lowest-cost green period. For a household with substantial daytime home-office demand, the tariff could still save money without a battery because daytime working consumption is mostly outside the 4pm–7pm peak. A battery could add value by charging overnight and covering peak consumption, but this source does not provide actual regional unit rates, standing charges, battery-compatible export/import terms, or battery economics. EDF’s “up to” discounts and £187 annual saving are promotional, based on October 2025 assumptions and dynamic prices; they should not be used as a direct July 2026 financial forecast without obtaining the current tariff quote and half-hourly consumption data.

**31. [S31] [Smart meter tariffs by E.ON Next | Use electricity when it is cheaper](https://www.eonnext.com/tariffs/smart-tariffs)**

For a UK household using 9,000 kWh/year, the page supports checking a time-of-use tariff before purchasing a battery. The household’s substantial daytime home-office demand may fit E.ON’s Smart Saver off-peak window of 5am–4pm, meaning meaningful savings may be available without storage if its quoted daytime rate is competitive. A battery could instead buy low-cost overnight electricity—illustratively 9p/kWh on E.ON Next Drive Fixed V24—and discharge during costly periods, but the page supplies no actual Smart Saver peak/off-peak prices or battery-compatible tariff terms for this household. Its 24.674p/kWh national-average comparator is close to the stated 24.7p/kWh flat rate, while the cited 26.11p/kWh July 2026 comparator and the supplier’s example demonstrate that outcomes are strongly determined by the fraction of consumption in cheap, off-peak, and 4–7pm peak periods. Obtain a postcode-specific quote and half-hourly smart-meter data before modelling savings; do not treat the advertised 8–9p/kWh EV overnight rates as automatically available or suitable for a non-EV battery household.

**32. [S32] [Energy prices are rising, should I fix my tariff?](https://www.eonnext.com/blog/energy-prices-are-rising-should-i-fix-my-tariff)**

For the no-battery baseline, the page supports checking competitive fixed or price-cap-linked tariffs rather than assuming the current £0.247/kWh flat rate is unavoidable. Next Pledge is described as a 12-month tracker that remains below the Ofgem SVT price cap, but its quoted £50 annual saving is for Ofgem’s average dual-fuel customer, varies by region and usage, and cannot be applied directly to this high-electricity-use household. Fixed tariffs can provide price certainty but may impose exit fees and prevent a switch to a potentially more valuable time-of-use tariff. The source confirms that the Energy Price Guarantee is no longer available, but offers no evidence sufficient for battery payback calculations or for selecting a time-of-use tariff.

**33. [S33] [Smart Export Guarantee (SEG) Tariffs | EDF Small Business](https://www.edfenergy.com/energywise/small-business-SEG-tariffs)**

For the stated England household with no solar or other eligible renewable generator, this source indicates that SEG should not be included as battery revenue: SEG concerns surplus electricity produced by qualifying renewable technologies, not electricity imported cheaply, stored in a battery, and later exported. The quoted 14–16p/kWh business export figures and EDF’s 15p/kWh business tariff are not appropriate inputs for the household’s battery-only financial model, and the page does not provide relevant domestic time-of-use import rates, battery installed costs, efficiency, degradation, cycling, or warranty evidence. Its useful contribution is limited to flagging that export remuneration is supplier-set, conditional, and potentially variable where applicable; a separate authoritative source is needed on whether a particular domestic tariff permits battery/grid arbitrage and export of imported electricity.

</details>
