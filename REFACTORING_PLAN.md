# Refactoring Plan: Soul Application to Reusable Module

**Goal:** Refactor the current application into a reusable Go module named `soul` that other applications can import, initialize, and extend with their own routes, while managing their own web server lifecycle.

**Key Changes:**

1.  **Restructure:** Move code from `internal/` to new top-level exported packages (`service`, `db`, `models`, `config`, `modules`, `jobs`, `web`, `core`). This includes moving `internal/handler` to `web/handlers` and `internal/middleware` to `web/middleware`.
2.  **Module API:**
    - Remove `package main` and `main()`.
    - Create a `soul` package.
    - Exported `config.Config` struct (defined in `config` package) for passing DB details, cache settings, enabled modules, etc. **Crucially, this will NOT contain web server host/port info.**
    - Exported `soul.New(cfg config.Config) (*service.Context, error)`: Initializes core services (DB, cache, modules, jobs based on `cfg`) and returns the `service.Context`. **Does NOT create a web server.**
    - **New Exported Function:** `soul.RegisterRoutes(e *echo.Echo, svcCtx *service.Context)` (or similar name, perhaps in the `web` package like `web.RegisterCoreRoutes`). This function takes an existing `*echo.Echo` instance (created by the consumer) and the `service.Context`, and registers all the core routes and necessary middleware defined within the `soul` module onto that instance.
3.  **Configuration Loading:** Remains the responsibility of the _consuming application_. It loads its config, populates `soul.Config`, and passes it to `soul.New`.
4.  **Module System:** Replace the implicit `init()`-based module registration with an _explicit_ mechanism. The `config.Config` will specify which modules to activate, and `soul.New` will initialize only those.
5.  **Web Layer Handling:**
    - The `soul` module **does not** create, configure (beyond adding its routes/middleware), or start the Echo web server.
    - The consuming application creates the `*echo.Echo` instance.
    - The consuming application applies its own middleware.
    - The consuming application registers its own custom routes.
    - The consuming application calls `soul.RegisterRoutes` (passing its Echo instance and the `service.Context` from `soul.New`) to add the core `soul` routes.
    - The consuming application starts and stops the Echo server.
6.  **Jobs:** Conditional initialization based on `soul.Config`.
7.  **Documentation:** Update README and examples to reflect this new initialization flow: Consumer creates Echo instance -> Consumer registers own routes -> Consumer calls `soul.RegisterRoutes` -> Consumer starts server.

**Visual Plan (Mermaid):**

```mermaid
graph TD
    subgraph Consumer App (main.go)
        direction LR
        C0[Load Own Config] --> C1[Populate soul.Config];
        C1 --> C2{Call soul.New(config)};
        C2 --> C3[Receive service.Context];
        C3 --> C4[Create Own Echo Instance];
        C4 --> C5[Apply Consumer Middleware];
        C5 --> C6[Register Consumer Routes];
        C6 --> C7{Call soul.RegisterRoutes(echoInstance, svcCtx)};
        C7 --> C8[Start Echo Server];
        C3 --> C9[Optionally Start Job Manager];
    end

    subgraph Soul Module (Exported Packages)
        direction LR
        S1[soul.New(config)] --> S2[Create service.Context];
        S2 --> S3[Initialize DB];
        S2 --> S4[Initialize Cache];
        S2 --> S5[Initialize Explicit Modules];
        S2 --> S8[Initialize Job Manager];
        S2 --> S9[Return service.Context];

        S10[soul.RegisterRoutes(e, svcCtx)] --> S11[Register Core Soul Routes on 'e'];
        S11 --> S12[Apply Core Soul Middleware on 'e'];


        P1[config/config.go] --> S1;
        P2[service/context.go] --> S2;
        P3[db/database.go] --> S3;
        P4[models/*] --> S3;
        P5[modules/init.go] --> S5;
        P7[jobs/manager.go] --> S8;
        P8[web/handlers/routes.go] --> S11; # Registers handlers from web/handlers/*
        P10[web/middleware/*] --> S12;
        P9[core/*] --> P8; # Logic used by handlers
    end

    C1 --> P1;
    C2 --> S1;
    S9 --> C3;
    C4 --> C7; # Pass Echo instance
    C3 --> C7; # Pass Service Context
    C7 --> S10; # Call the registration function
    C9 --> S8;


    style Consumer App fill:#f9f,stroke:#333,stroke-width:2px;
    style Soul Module fill:#ccf,stroke:#333,stroke-width:2px;
```
