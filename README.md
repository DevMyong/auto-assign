# auto-assign-defaults

**auto-assign-defaults** is a GitHub Action that automatically assigns default metadata to pull requests. If a PR is
missing labels, assignees, or reviewers, this Action will add defaults based on predefined settings and even add
specific labels based on keywords in the PR title.

---

## Features

- **Title-Based Label Assignment:**  
  Automatically adds labels based on the PR title:
    - If the title contains `feat`, the label `enhancement` is added.
    - If the title contains `fix`, the label `bug` is added.

- **Dynamic D-n Labeling:**  
  In addition to title-based labeling, the Action dynamically assigns a D-n label based on the size of the code changes:
  - For small code changes, a lower D-n value (e.g., `D-3`) is applied.
  - For large code changes, a higher D-n value (e.g., `D-5`) is applied. 
  (ex. 400 is the threshold for determining the size of the code changes.)

- **Default Assignee:**  
  The PR author is automatically set as the default assignee.

- **Default Reviewer Assignment:**  
  By default, all contributors of the repository are considered as potential reviewers.  
  If there are more than 10, a random selection of 10 reviewers is made.

- **Consistent PR Process:**  
  Helps prevent oversights during manual PR creation by ensuring critical review steps are never missed.

---

## Usage

To integrate **auto-assign-defaults** into your workflow, add the following step to your GitHub Actions workflow file (
e.g., `.github/workflows/ci-pr-open.yml`):

```yaml
name: PR Open Automation

on:
  pull_request:
    types: [opened, ready_for_review]

jobs:
  add-auto-assign:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v3

      - name: Assigns default PR metadata
        uses: devmyong/auto-assign@v1.0.0
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          PR_NUMBER: ${{ github.event.number }}
```

---

## Explanation

To ensure consistent and effective pull request management, this Action relies on clear conventions and dynamic criteria:

1. **PR Title Conventions:**  
   It is essential to follow a standardized PR title format. For example:
   - Use `feat: ...` when introducing new features.
   - Use `fix: ...` when addressing bug fixes.  
    
   This standardization allows the Action to automatically determine and apply the appropriate label (`enhancement` or `fix`) based on the title content.  
   Follow the mapping of keywords to labels:

    ```
   "feat":     "enhancement",
   "fix":      "bug",
   "docs":     "documentation",
   "style":    "style",
   "refactor": "refactor",
   "perf":     "performance",
   "test":     "test",
   "chore":    "chore",
   ```

2. **Dynamic `D-n` Labeling Based on Code Size:**  
   The Action evaluates the magnitude of code changes in the pull request. Depending on the size:
   - A smaller change set results in a lower `D-3` value, indicating that minimal review effort is required.
   - A larger change set results in a higher `D-5` value, suggesting that more extensive review and testing may be necessary.  
   
   This dynamic labeling helps prioritize the review process according to the complexity of the changes.

3. **Default Reviewer and Assignee Strategy:**  
   The Action automatically assigns the PR author as the default assignee. For reviewers, it considers all contributors of the repository. When the number of contributors exceeds 10, a random subset of 10 reviewers is chosen.  
   This approach helps streamline the review process by ensuring a manageable number of reviewers are requested while still covering a broad range of expertise.

---
