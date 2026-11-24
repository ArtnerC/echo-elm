// Chains of set operators. CQF regroups a union chain, and an intersect chain,
// "right-leaning bushy": a ∪ b ∪ c ∪ d is (a ∪ b) ∪ (c ∪ d), with the regrouped
// inner node carrying no locator or result type. Three-way chains and except
// chains stay left-deep. (issues/08 M3, LibraryBuilder.resolveUnion)
library SetOperationChains version '1.0'

using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers

parameter "Period" Interval<DateTime>

context Patient

define "L1": { 1, 2 }
define "L2": { 2, 3 }
define "L3": { 3, 4 }
define "L4": { 4, 5 }
define "L5": { 5, 6 }

define "Union of three":
  "L1"
    union "L2"
    union "L3"

define "Union of four":
  "L1"
    union "L2"
    union "L3"
    union "L4"

define "Union of five":
  "L1"
    union "L2"
    union "L3"
    union "L4"
    union "L5"

define "Pipe of four": "L1" | "L2" | "L3" | "L4"

define "Intersect of four":
  "L1"
    intersect "L2"
    intersect "L3"
    intersect "L4"

define "Except of four":
  "L1"
    except "L2"
    except "L3"
    except "L4"

define "Interval intersect of four":
  "Period"
    intersect "Period"
    intersect "Period"
    intersect "Period"

define "Parenthesized":
  ("L1" union "L2")
    union ("L3" union "L4")

define "Retrieves of four":
  [Encounter]
    union [Encounter]
    union [Encounter]
    union [Encounter]
