# Schemer Error Report

### Code: `0001`
**Func Name:** `parseApplyCommand`

**Message:** --conn-key or --conn-string must be used.

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/apply.go:102:10`

---

### Code: `0002`
**Func Name:** `parseApplyCommand`

**Message:** ...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/apply.go:112:11`

---

### Code: `0003`
**Func Name:** `parseApplyCommand`

**Message:** flags --from/--to cannot be used with --cherry-pick

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/apply.go:121:10`

---

### Code: `0004`
**Func Name:** `CreateSchemerTable`

**Message:** recived nil database pointer.

**Location:** `/home/inskribe/dev/go/schemer/internal/utils/database.go:103:10`

---

### Code: `0005`
**Func Name:** `CreateSchemerTable`

**Message:** failed to read schemer.sql from deltas directory.

**Location:** `/home/inskribe/dev/go/schemer/internal/utils/database.go:115:10`

---

### Code: `0006`
**Func Name:** `CreateSchemerTable`

**Message:** failed to query for table during pre-creation check.

**Location:** `/home/inskribe/dev/go/schemer/internal/utils/database.go:131:10`

---

### Code: `0007`
**Func Name:** `CreateSchemerTable`

**Message:** failed to create schemer table.

**Location:** `/home/inskribe/dev/go/schemer/internal/utils/database.go:145:10`

---

### Code: `0008`
**Func Name:** `WithConn`

**Message:** failed to conntect with database.

**Location:** `/home/inskribe/dev/go/schemer/internal/utils/database.go:81:10`

---

### Code: `0009`
**Func Name:** `ConnectDatabase`

**Message:** failed to establishe a connection with database.

**Location:** `/home/inskribe/dev/go/schemer/internal/utils/database.go:50:15`

---

### Code: `0010`
**Func Name:** `ConnectDatabase`

**Message:** failed to ping database.

**Location:** `/home/inskribe/dev/go/schemer/internal/utils/database.go:58:15`

---

### Code: `0011`
**Func Name:** ``

**Message:** failed to get current working.

**Location:** `/home/inskribe/dev/go/schemer/internal/utils/directory.go:41:14`

---

### Code: `0012`
**Func Name:** `LoadDotEnv`

**Message:** failed to load execute godotenv.Load

**Location:** `/home/inskribe/dev/go/schemer/internal/utils/env.go:55:17`

---

### Code: `0013`
**Func Name:** `WriteEnvFile`

**Message:** failed to parse env template.

**Location:** `/home/inskribe/dev/go/schemer/internal/templates/env.go:45:10`

---

### Code: `0014`
**Func Name:** `WriteEnvFile`

**Message:** failed to create .env file.

**Location:** `/home/inskribe/dev/go/schemer/internal/templates/env.go:54:10`

---

### Code: `0015`
**Func Name:** `WriteEnvFile`

**Message:** failed to wirte template to .env file.

**Location:** `/home/inskribe/dev/go/schemer/internal/templates/env.go:62:10`

---

### Code: `0016`
**Func Name:** `WriteTemplate`

**Message:** failed to parse template args.

**Location:** `/home/inskribe/dev/go/schemer/internal/templates/schemer.go:44:10`

---

### Code: `0017`
**Func Name:** `WriteTemplate`

**Message:** failed to create schemer.sql file in the deltas directory.

**Location:** `/home/inskribe/dev/go/schemer/internal/templates/schemer.go:54:10`

---

### Code: `0018`
**Func Name:** `WriteTemplate`

**Message:** failed to parse schemer template.

**Location:** `/home/inskribe/dev/go/schemer/internal/templates/schemer.go:63:10`

---

### Code: `0019`
**Func Name:** `Parse`

**Message:** detected illegal character in table name

**Location:** `/home/inskribe/dev/go/schemer/internal/templates/schemer.go:76:10`

---

### Code: `0020`
**Func Name:** `getAppliedDeltas`

**Message:** expected pointer to pgx.Conn, recived nil

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/helpers.go:210:15`

---

### Code: `0021`
**Func Name:** `getAppliedDeltas`

**Message:** `failed to find schemer table. Schemer table is used to track migrations and must be present.
Ensure project was setup with [schemer] init`

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/helpers.go:222:17`

---

### Code: `0022`
**Func Name:** `getAppliedDeltas`

**Message:** failed to query applied versions

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/helpers.go:231:15`

---

### Code: `0023`
**Func Name:** `getAppliedDeltas`

**Message:** failed to scan version

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/helpers.go:245:16`

---

### Code: `0024`
**Func Name:** `getAppliedDeltas`

**Message:** row iteration error:

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/helpers.go:256:15`

---

### Code: `0025`
**Func Name:** `getRequestedDeltas`

**Message:** invalid cherry-picked tag: ...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/helpers.go:184:16`

---

### Code: `0026`
**Func Name:** `getRequestedDeltas`

**Message:** failed to convert --to tag...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/helpers.go:166:16`

---

### Code: `0027`
**Func Name:** `getRequestedDeltas`

**Message:** failed to convert --from tag...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/helpers.go:154:16`

---

### Code: `0028`
**Func Name:** `executeDownCommand`

**Message:** failed to apply delta: ...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/down.go:135:17`

---

### Code: `0029`
**Func Name:** `executeDownCommand`

**Message:** failed to update schemer table

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/down.go:152:16`

---

### Code: `0030`
**Func Name:** `loadDownDeltas`

**Message:** expected valid deltaRequest, recieved nil

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/down.go:177:15`

---

### Code: `0031`
**Func Name:** `loadDownDeltas`

**Message:** failed to read directory from path: ...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/down.go:190:15`

---

### Code: `0032`
**Func Name:** `loadDownDeltas`

**Message:** malformed filename: ...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/down.go:211:16`

---

### Code: `0033`
**Func Name:** `loadDownDeltas`

**Message:** failed to read file at path: ...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/down.go:226:17`

---

### Code: `0034`
**Func Name:** `loadDownDeltas`

**Message:** failed to read file at path: ...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/down.go:253:16`

---

### Code: `0035`
**Func Name:** `applyForLastUpDelta`

**Message:** There are no applied deltas in the schemer table, Aborting apply last delta.

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/down.go:282:10`

---

### Code: `0036`
**Func Name:** `applyForLastUpDelta`

**Message:** failed to find down delta for last applied up delta: ...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/down.go:304:10`

---

### Code: `0037`
**Func Name:** `applyForLastUpDelta`

**Message:** failed to apply delta: ...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/down.go:315:16`

---

### Code: `0038`
**Func Name:** `applyForLastUpDelta`

**Message:** failed to update schemer table.

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/down.go:329:15`

---

### Code: `0039`
**Func Name:** `fetchPostStatuses`

**Message:** failed to execute select query for schemer table.

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/post.go:92:15`

---

### Code: `0041`
**Func Name:** `fetchPostStatuses`

**Message:** failed to scan row

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/post.go:106:16`

---

### Code: `0042`
**Func Name:** `fetchPostStatuses`

**Message:** iteration failure on pgx.Rows

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/post.go:116:15`

---

### Code: `0043`
**Func Name:** `loadPostDeltas`

**Message:** failed to read directory at: ...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/post.go:162:15`

---

### Code: `0044`
**Func Name:** `loadPostDeltas`

**Message:** deltas directory is empty.

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/post.go:169:15`

---

### Code: `0045`
**Func Name:** `loadPostDeltas`

**Message:** malformed delta tag: ...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/post.go:192:16`

---

### Code: `0046`
**Func Name:** `loadPostDeltas`

**Message:** failed to read delta file at: ...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/post.go:215:16`

---

### Code: `0047`
**Func Name:** `executePostCommand`

**Message:** failed to apply post delta: ...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/post.go:270:14`

---

### Code: `0048`
**Func Name:** `executePostCommand`

**Message:** failed to update schemer table.

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/post.go:291:15`

---

### Code: `0049`
**Func Name:** `loadUpDeltas`

**Message:** failed to read directory at: ...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/up.go:88:15`

---

### Code: `0050`
**Func Name:** `loadUpDeltas`

**Message:** malformed delta tag...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/up.go:110:16`

---

### Code: `0051`
**Func Name:** `loadUpDeltas`

**Message:** failed to read file at: ...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/up.go:134:16`

---

### Code: `0052`
**Func Name:** `applyUpDeltas`

**Message:** all requested deltas have been already applied.

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/up.go:220:10`

---

### Code: `0053`
**Func Name:** `applyUpDeltas`

**Message:** failed to apply delta: ...

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/up.go:248:14`

---

### Code: `0054`
**Func Name:** `applyUpDeltas`

**Message:** failed to update schemer table.

**Location:** `/home/inskribe/dev/go/schemer/cmd/apply/up.go:269:15`

---

### Code: `0055`
**Func Name:** `determineNextTag`

**Message:** malformed delta tag: ...

**Location:** `/home/inskribe/dev/go/schemer/cmd/create/create.go:124:15`

---

### Code: `0056`
**Func Name:** `determineNextTag`

**Message:** failed to read directory at: ...

**Location:** `/home/inskribe/dev/go/schemer/cmd/create/create.go:96:14`

---

### Code: `0057`
**Func Name:** `createDeltaFiles`

**Message:** ......

**Location:** `/home/inskribe/dev/go/schemer/cmd/create/create.go:164:10`

---

### Code: `0058`
**Func Name:** `createDeltaFiles`

**Message:** ......

**Location:** `/home/inskribe/dev/go/schemer/cmd/create/create.go:171:10`

---

### Code: `0059`
**Func Name:** `createDeltaFiles`

**Message:** failed to create up.sql file

**Location:** `/home/inskribe/dev/go/schemer/cmd/create/create.go:179:10`

---

### Code: `0060`
**Func Name:** `createDeltaFiles`

**Message:** failed to create down.sql file

**Location:** `/home/inskribe/dev/go/schemer/cmd/create/create.go:190:10`

---

### Code: `0061`
**Func Name:** `createDeltaFiles`

**Message:** ......

**Location:** `/home/inskribe/dev/go/schemer/cmd/create/create.go:213:10`

---

### Code: `0062`
**Func Name:** `createDeltaFiles`

**Message:** failed to create post.sql file

**Location:** `/home/inskribe/dev/go/schemer/cmd/create/create.go:221:10`

---

### Code: `0063`
**Func Name:** `createDeltasDirectory`

**Message:** failed to create deltas directory.

**Location:** `/home/inskribe/dev/go/schemer/cmd/init/init.go:166:10`

---

### Code: `0064`
**Func Name:** `createEnvFile`

**Message:** empty working directory.

**Location:** `/home/inskribe/dev/go/schemer/cmd/init/init.go:190:10`

---

### Code: `0065`
**Func Name:** `createEnvFile`

**Message:** Schemer does not support .env directory

**Location:** `/home/inskribe/dev/go/schemer/cmd/init/init.go:201:10`

---

### Code: `0066`
**Func Name:** `createDeltasDirectory`

**Message:** empty directory path

**Location:** `/home/inskribe/dev/go/schemer/cmd/init/init.go:153:10`

---

### Code: `0067`
**Func Name:** `executeInitCommand`

**Message:** failed to get working directory.

**Location:** `/home/inskribe/dev/go/schemer/cmd/init/init.go:86:10`

---

