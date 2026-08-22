# Adversarial review

Every substantive change gets an adversarial review by someone who did not
write it, AFTER the commits are done and BEFORE the pull request is opened.
Fix the findings, re-gate, then open the PR carrying the verdict. The coverage
gate proves the code does what its tests say; the review finds the tests
asking the wrong questions.

## The briefing (what the review must be given)

A reviewer without context can only lint. Every review brief carries:

- **Why these commits exist**: the motivating spec rule or issue, in one or two
  sentences, with the rule ID (G01-G13) or issue number.
- **The problem**: what is missing or wrong, with concrete evidence.
- **The intended solution**: how the change is SUPPOSED to solve the problem,
  mechanism by mechanism, as the author understands it.
- **The riskiest surfaces**: where the author thinks it can break (false
  positives on idiomatic Go, comment-position edge cases, type-info gaps in
  fixtures, cross-package contracts). Authors self-nominate risks in their
  reports; pass them through.

## The mandate (what the review judges)

Two questions, both mandatory:

1. **Is the code right?** Trace claims end to end against the actual code.
   Execute probes where possible: run the analyzer on a counterexample
   fixture, write the failing test, check the diagnostic position. A finding
   states file, claim, and a concrete failure scenario, marked CONFIRMED
   (traced or reproduced) or PLAUSIBLE (needs the author). Findings ranked by
   severity.
2. **Is this the right solution?** Judge the fit against the spec entry, not
   just the diff: does it enforce the stated rule or a nearby easier one?
   Does it fire on idiomatic Go the spec exempts? Is there a smaller check
   (KISS)? Does it build machinery the rule does not need (YAGNI)? Should the
   rule exist at all? A technically clean analyzer that answers the wrong
   question is NOT READY.

## The verdict

Exactly one of: **READY FOR PR** / **READY AFTER FIXES** (listed, required
vs optional) / **NOT READY** (with the reasoning that reopens the design).
Honest nulls are valuable: "no findings survived verification" must mean the
reviewer attacked and failed, not that they skimmed.

## Mechanics

- The reviewer is impartial: a separate agent or person, never the author,
  read-only, opens no PR, merges nothing, posts nothing to GitHub.
- Reviews run against the pushed branch; note (do not fix) rebase needs.
- The PR body records the verdict and what the fixes addressed, so merge
  history carries the review provenance.
