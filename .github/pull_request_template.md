## Pull Request Title: [Brief, Descriptive Title]

**Description:**

_Please provide a detailed description of the changes in this pull request. What problem does it solve? What new features does it add?_

---

### Change-Specific Checklist

_Please check off the sections below that are relevant to this PR and complete the corresponding details._

**Domain Changes**

- [ ] This PR introduces changes to the domain models or business logic.
  - **List of Changed Domains:**
    - `[Domain 1]`
    - `[Domain 2]`

---

**Database Migration**

- [ ] This PR includes a database migration.
  - **Table Changes:**
    - _e.g., `users` table: Added `last_login_at` column._
    - _e.g., `products` table: Removed `old_price` column._
  - **Migration Strategy:**
    - _e.g., Adding a non-nullable column: Is a default value provided?_
    - _e.g., Dropping a column: Is the data backed up?_

---

**Endpoint Changes**

- [ ] This PR adds, modifies, or removes API endpoints.
  - **List of Changed Endpoints:**
    - `GET /api/v1/new-resource` (New)
    - `PUT /api/v1/old-resource/{id}` (Modified)
  - [ ] **Swagger/API Docs** have been updated for all changed endpoints.
  - [ ] **Breaking Change:** Does this PR introduce a breaking change to an existing API? (Yes/No)
    - If yes, please explain the impact: _[Explanation]_