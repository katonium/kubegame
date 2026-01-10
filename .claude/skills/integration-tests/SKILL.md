---
name: integration-tests
description: Use this skill when creating, updating, or debugging API contract tests and Scenario workflow tests using Playwright. Triggers on requests involving test creation, test fixtures, API testing, workflow testing, or test debugging.
---

# Integration Tests (API + Scenario)

## Goal

Standardize the creation and maintenance of API contract tests and Scenario workflow tests using Playwright, ensuring clear separation of concerns and maintainable test suites.

## Test Type Decision

**Before writing any test, ask:**
> "Am I verifying a contract or a workflow?"

| Answer | Test Type | Directory |
|--------|-----------|-----------|
| Contract (single API correctness) | API Test | `tests/api/` |
| Workflow (multi-API behavior) | Scenario Test | `tests/scenario/` |
| User experience (UI-driven) | E2E Test | `tests/e2e/` |

## Instructions

### 1. Creating an API Test

API tests verify **API contracts** - single endpoint behavior in isolation.

**Step 1:** Determine the endpoint and HTTP method
**Step 2:** Create file in `tests/api/{resource}/{method}-{resource}.spec.ts`
**Step 3:** Import from API fixtures: `import { test, expect, apiAssertions } from "../../fixtures/api";`
**Step 4:** Write tests using **Given-When-Then** structure with `test.step()`:
- **Given** - Setup preconditions (create test data, prepare context)
- **When** - Execute the action being tested (make API call)
- **Then** - Verify the results (assert status, response fields, side effects)

**Test Structure:**
```typescript
test("description", async ({ fixtures }) => {
  let response: any;
  let testData: any;

  await test.step("Given [precondition]", async () => {
    // Setup: create data, prepare context
  });

  await test.step("When [action]", async () => {
    // Action: make the API call
    response = await apiContext.method("/endpoint");
  });

  await test.step("Then [expected result]", async () => {
    await test.step("With status code X", async () => {
      // Assert status code
    });

    await test.step("And [additional verification]", async () => {
      // Assert response fields, side effects
    });
  });
});
```

**What to verify:**
- Status codes (200, 201, 400, 401, 404)
- Response schema (required fields, types)
- Input validation (missing/invalid fields)
- Authorization enforcement (401, 404 for security)
- Side effects (database changes, if applicable)

**Example (Given-When-Then with test.step):**
```typescript
// tests/api/album/v1/albums/get-album-detail.spec.ts
import { test, expect, apiAssertions } from "../../../../fixtures/api";

test.describe("GET /api/v1/album/albums/{album_key}", () => {
  test("returns 200 with album details when user is the owner", async ({
    hostApiContext,
    createMinimalTestData,
    logger
  }) => {
    let response: any;
    let responseBody: any;
    let albumData: any;

    await test.step("Given an album created by the authenticated user", async () => {
      albumData = createMinimalTestData("album");
      const createResponse = await hostApiContext.post("/api/v1/album/albums", {
        data: {
          name: albumData.name,
          default_group_name: "All Guests",
          display_name: "Owner Name",
          event_date: "2025-06-15",
        },
      });
      responseBody = await createResponse.json();
      logger.info("Created album as owner", { album_key: responseBody.album_key });
    });

    await test.step("When the owner requests album details", async () => {
      response = await hostApiContext.get(
        `/api/v1/album/albums/${responseBody.album_key}`
      );
    });

    await test.step("Then the API returns the album details", async () => {
      await test.step("With status code 200", async () => {
        apiAssertions.hasStatus(response, 200);
        await apiAssertions.isJson(response);
      });

      await test.step("And the response includes all required fields", async () => {
        const data = await response.json();
        expect(data).toMatchObject({
          album_key: responseBody.album_key,
          name: albumData.name,
          event_date: "2025-06-15",
          membership: {
            role: "owner",
            display_name: "Owner Name",
          },
        });
        logger.info("Verified owner can access album details", {
          role: data.membership.role,
        });
      });
    });
  });

  test("returns 401 when no authentication token is provided", async ({
    envConfig,
    logger
  }) => {
    let response: any;

    await test.step("Given an unauthenticated request", async () => {
      logger.info("Creating request without auth token");
    });

    await test.step("When the client requests album details without authentication", async () => {
      response = await fetch(
        `${envConfig.baseUrl}/api/v1/album/albums/fake-album-key`
      );
    });

    await test.step("Then the API returns 401 Unauthorized", async () => {
      expect(response.status).toBe(401);
      logger.info("Correctly rejected unauthenticated request", {
        status: response.status,
      });
    });
  });
});
```

**Given-When-Then Best Practices:**

1. **Variable Declaration** - Declare response/data variables at test function scope
   ```typescript
   test("description", async ({ fixtures }) => {
     let response: any;
     let albumKey: string;
     // Then use in steps
   });
   ```

2. **Given Steps** - Setup only what's necessary for THIS test
   - Create minimal test data
   - Log setup actions
   - Don't test anything in Given (just setup)

3. **When Steps** - Single action being tested
   - Make ONE primary API call
   - This is the behavior under test
   - Keep it simple and focused

4. **Then Steps** - Comprehensive verification
   - Always verify status code first (nested step)
   - Then verify response structure/fields
   - Then verify side effects (database, if needed)
   - Use nested steps for clarity

5. **Step Naming Conventions:**
   - "Given [context/precondition]"
   - "When [action being tested]"
   - "Then [expected outcome]"
   - "With [specific detail]"
   - "And [additional verification]"

**Allowed in API Tests:**
- ✅ Validate status codes
- ✅ Validate response schemas
- ✅ Test required/optional fields
- ✅ Test authorization errors (401/403/404)
- ✅ Use minimal test data
- ✅ Use API-specific fixtures
- ✅ Use test.step() for Given-When-Then structure
- ✅ Verify database side effects (if needed)

**Forbidden in API Tests:**
- ❌ Calling multiple APIs to prepare state (use database instead)
- ❌ Testing business workflows
- ❌ Depending on execution order
- ❌ Using Scenario fixtures
- ❌ Creating complex relational data
- ❌ Asserting internal implementation details
- ❌ Mixing test assertions with setup code

### 2. Creating a Scenario Test

Scenario tests verify **business workflows** - multiple APIs working together.

**Step 1:** Identify the user workflow to test
**Step 2:** Create file in `tests/scenario/{domain}/{workflow-name}.spec.ts`
**Step 3:** Import from Scenario fixtures: `import { test, expect, workflowAssertions } from "../../fixtures/scenario";`
**Step 4:** Write tests that verify:
- End-to-end workflow completion
- State transitions
- Cross-entity behavior
- Permission flows

**Example:**
```typescript
// tests/scenario/album/create-album.spec.ts
import { test, expect, workflowAssertions } from "../../fixtures/scenario";

test.describe("Album Creation Workflow", () => {
  test("host can create album, add groups, and generate share links", async ({
    hostContext,
    testId,
  }) => {
    // Step 1: Create album
    const albumResponse = await hostContext.apiContext.post("/api/albums", {
      data: { name: `Wedding Album ${testId}` },
    });
    await workflowAssertions.stepSucceeded(async () => albumResponse);
    const album = await albumResponse.json();

    // Step 2: Add group
    const groupResponse = await hostContext.apiContext.post(
      `/api/albums/${album.id}/groups`,
      { data: { name: "Groom's Friends" } }
    );
    await workflowAssertions.stepSucceeded(async () => groupResponse);

    // Step 3: Verify final state
    const detailResponse = await hostContext.apiContext.get(`/api/albums/${album.id}`);
    const detail = await detailResponse.json();
    expect(detail.groups).toHaveLength(1);

    // Cleanup
    await hostContext.apiContext.delete(`/api/albums/${album.id}`);
  });
});
```

**Allowed in Scenario Tests:**
- ✅ Call multiple APIs in sequence
- ✅ Reuse intermediate states
- ✅ Create realistic test data
- ✅ Use database seeding if needed
- ✅ Use Scenario-specific fixtures
- ✅ Focus on "does the flow work"

**Forbidden in Scenario Tests:**
- ❌ Testing detailed validation rules
- ❌ Verifying every response field
- ❌ Testing a single API in isolation
- ❌ UI interactions (belongs to E2E)
- ❌ Low-level schema assertions

### 3. Fixture Usage Rules

```
core fixtures (stateless)
    ↓
api fixtures        scenario fixtures
(minimal setup)     (rich setup)
```

**Core Fixtures** (`tests/fixtures/core/`):
- HTTP client setup, Base URL, Headers
- Environment loading
- Logging and tracing
- Test ID generation
- **Can be used everywhere**

**API Fixtures** (`tests/fixtures/api/`):
- Minimal test users
- Single-entity setup
- Lightweight, stateless data
- **Only for API tests**

**Scenario Fixtures** (`tests/fixtures/scenario/`):
- Database seeding
- Multi-entity relationships
- Pre-built intermediate states
- Cleanup logic
- **Only for Scenario tests**

### 4. Absolute Rules (Non-Negotiable)

| Rule | Consequence of Breaking |
|------|------------------------|
| ❌ Scenario fixtures MUST NOT be used in API tests | Tests become slow and coupled |
| ✅ Core fixtures MAY be used everywhere | Safe to share |
| ❌ API tests MUST NOT encode workflows | Unclear test responsibility |
| ❌ Scenario tests MUST NOT verify API contracts | Redundant coverage |
| ❌ Tests MUST NOT depend on execution order | Flaky in parallel |

### 5. Running Tests

```bash
# All tests
pnpm test
# or
task test

# API tests only
pnpm test:api
# or
task test:api

# Scenario tests only
pnpm test:scenario
# or
task test:scenario

# With UI
pnpm test:ui
# or
task test:ui

# Specific file
pnpm exec playwright test tests/api/albums/get-albums.spec.ts

# With filter
pnpm exec playwright test -g "should return 200"
```

### 6. Adding New Fixtures

**For Core fixtures:**
1. Add to `tests/fixtures/core/utils.ts` for utility functions
2. Extend `CoreFixtures` interface in `tests/fixtures/core/index.ts`
3. Add fixture in the `test.extend()` call

**For API fixtures:**
1. Add to `tests/fixtures/api/index.ts`
2. Keep stateless and minimal
3. Never import from Scenario fixtures

**For Scenario fixtures:**
1. Add to `tests/fixtures/scenario/index.ts`
2. Include cleanup logic
3. Never import into API fixtures

## Constraints

- Each API test file tests exactly ONE endpoint
- Scenario tests must include cleanup (automatic via fixtures or explicit)
- Never hard-code URLs; use `envConfig.baseUrl`
- Always use `testId` for test data isolation
- Use `logger` for debugging, not `console.log`
- All tests must pass independently and in parallel

## Security & Authorization Patterns

**Anonymous User Access:**
- Anonymous users (guests) should have LIMITED access
- GET endpoints for listing user-owned resources should return 401 for anonymous users
- GET endpoints for accessing shared resources (by key) should return 200 if the anonymous user is a member

**Authorization Error Responses:**
- Return **404** instead of **403** when a user tries to access a resource they don't own
- This prevents leaking information about resource existence
- Only return 403 for clearly documented permission errors

**Response Validation:**
- Assert response fields directly in each test case
- Don't create separate "response validation" tests
- Verify response structure as part of the success test

**Example:**
```typescript
test("returns 200 with album details when user is owner", async ({
  hostApiContext,
  createMinimalTestData
}) => {
  // Create album
  const albumData = createMinimalTestData("album");
  const createResponse = await hostApiContext.post("/api/v1/album/albums", {
    data: {
      name: albumData.name,
      default_group_name: "All Guests",
      display_name: "Host",
      event_date: "2025-12-31"
    }
  });
  const { album_key } = await createResponse.json();

  // Get album details
  const response = await hostApiContext.get(`/api/v1/album/albums/${album_key}`);

  // Verify status and response structure
  apiAssertions.hasStatus(response, 200);
  const data = await response.json();

  // Assert all required fields in THIS test
  expect(data).toMatchObject({
    album_key: expect.any(String),
    name: expect.any(String),
    event_date: expect.any(String),
    membership: {
      role: "owner",
      display_name: expect.any(String)
    }
  });
});

test("returns 404 when user is not a member (not 403)", async ({
  host1ApiContext,
  host2ApiContext
}) => {
  // Host1 creates album
  const createResponse = await host1ApiContext.post("/api/v1/album/albums", {
    data: { name: "Private Album", ... }
  });
  const { album_key } = await createResponse.json();

  // Host2 tries to access - should get 404, NOT 403
  const response = await host2ApiContext.get(`/api/v1/album/albums/${album_key}`);
  apiAssertions.isNotFound(response); // Returns 404 for security
});
```

## Project Structure Reference

```
tests/
├── api/                    # API contract tests
│   └── ws/
│       └── apiname/
│           ├── function.spec.ts         # tests for function1
│           └── function.spec.ts         # tests for function2
├── scenario/               # Business workflow tests
│   ├── scenario-group/     # Scenario group like `create-album`
│   │   └── specific-scenario.spec.ts
├── e2e/                    # UI-driven tests (future)
├── fixtures/
│   ├── core/
│   │   ├── index.ts        # Core test fixtures
│   │   ├── auth.ts         # Authentication utilities
│   │   └── utils.ts        # Utility functions
│   ├── api/
│   │   └── index.ts        # API test fixtures
│   └── scenario/
│       └── index.ts        # Scenario test fixtures
├── scripts/                # Test helper scripts if needed
├── global-setup.ts
├── global-teardown.ts
├── .env.template
└── README.md
```

## Examples

### User Input
"Create an API test for the POST /api/groups endpoint"

### Agent Output
1. Create file `tests/api/groups/post-groups.spec.ts`
2. Import from `../../fixtures/api`
3. Write tests for:
   - 201 on successful creation
   - 400 on validation errors
   - 401 on unauthenticated access
   - 404 when album doesn't exist
4. Use `createMinimalTestData("group")` for test data

### User Input
"Create a scenario test for guest joining and uploading photos"

### Agent Output
1. Create file `tests/scenario/guest/join-and-upload.spec.ts`
2. Import from `../../fixtures/scenario`
3. Use `seedAlbumWithGroups()` to create test data
4. Write workflow steps: join group → upload photo → verify visibility
5. Use `workflowAssertions.stepSucceeded()` for each step
