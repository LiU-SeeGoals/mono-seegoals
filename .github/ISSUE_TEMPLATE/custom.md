name: Software issue
description: File a new issue.
title: "[SW-AI]: "
labels: ["bug", "triage"]
projects: ["octo-org/1", "octo-org/44"]
assignees:
  - octocat
type: bug
body:
  - type: markdown
    attributes:
      value: |
        Thanks for taking the time to fill out this bug report!
  - type: input
    id: contact
    attributes:
      label: Contact Details
      description: How can we get in touch with you if we need more info?
      placeholder: please provide your discord ID
    validations:
      required: true
  - type: textarea
    id: structure
    attributes:
      label: Describe the issue:
      description: Also tell us, what did you expect to happen?
      placeholder: Tell us what you see!
      value: "A bug happened!"
    validations:
      required: true
  - type: dropdown
    id: time
    attributes:
      label: How long do you estimate to implement this?
      multiple: false
      options:
        - 1 week
        - 2 weeks
        - 3 weeks
        - 4 or more weeks
