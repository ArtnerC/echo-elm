// Result types on list-shaped operators, property navigation, and singleton from.
//
// Four separate defects converged on one symptom, List<Any> (issues/05 #12):
//   - Distinct/Flatten are UnaryExpression in ELM, so they never reached the
//     n-ary branch that follows the operand, and carried no result type.
//   - A property off a query alias had no lookup at all: the alias' element type
//     is a model type, not a tuple.
//   - Inherited FHIR elements — Condition.id is declared on Resource — were
//     missing from the property table, which does not walk baseType.
//   - SingletonFrom has its own ELM node and reached no operator branch.
//
// An untyped node is not locally visible: it silently degrades every enclosing
// expression's type, which is why these presented as one collapsed list type.
library ResultTypeInference version '1.0'

using FHIR version '4.0.1'

context Patient

define "Encs": [Encounter]
define "Dates": { @2024-01-01, @2024-06-01 }

// List-shape-preserving unary operators.
define "Distinct Dates": distinct "Dates"
define "Flat Dates": flatten { { @2024-01-01 }, { @2024-02-01 } }

// Property navigation off a query alias: complex type, and an inherited element.
define "Subjects": "Encs" E return E.subject
define "Ids": "Encs" E return E.id

// singleton from over a query.
define "Only One": singleton from ("Encs" E where E.id is not null)
