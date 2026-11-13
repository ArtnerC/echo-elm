library AgeFunctions version '1.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

context Patient

// Direct calls to the system age functions are CalculateAgeAt{precision}.
define "Age in months":
  CalculateAgeInMonthsAt(Patient.birthDate, Today())

define "Age in years":
  CalculateAgeInYearsAt(Patient.birthDate, Today())

// Control: the AgeInYearsAt form.
define "Age in years at":
  AgeInYearsAt(Today())
