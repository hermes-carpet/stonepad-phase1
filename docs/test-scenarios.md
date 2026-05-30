# Manual Test Scenarios

Documented test scenarios for v1 acceptance testing. See §11.3 of the
Stonepad v1 Implementation Plan.

## 1. Create Note, Sync, Verify on Server

1. Open Stonepad app
2. Tap **+** → Create a new note named `test-sync`
3. Type markdown content: `# Hello\n\nThis is a test note.`
4. Wait ~7 seconds for debounce auto-save
5. Tap **Sync Now** or wait for next poll cycle
6. Check server: `curl http://<server>/api/v1/notes/test-sync.md`
7. **Expected:** Server returns the note content exactly as typed

## 2. Edit Note Offline, Toggle Online, Verify Sync

1. Create a note while online → verify it syncs
2. Open Settings → toggle **Offline mode** ON
3. Edit the note (add content)
4. Toggle **Offline mode** OFF
5. Tap **Sync Now**
6. Check server for the updated note
7. **Expected:** Changes appear on server after coming back online

## 3. Edit Same Note on Two Devices, Verify Conflict Handling

1. Device A: Create `conflict-test.md` with content "Version from A"
2. Wait for sync to complete
3. Device B: Pull the note, edit to "Version from B", sync
4. Device A: Edit the note to "Version from A (edited)", sync
5. **Expected:** Conflict detected. Server version saved to `conflicts/` folder.
   Local version preserved. Note marked `conflict_pending`.

## 4. Disconnect Network Mid-Sync, Verify Graceful State

1. Start a sync cycle (tap Sync Now)
2. Immediately disconnect network (airplane mode)
3. **Expected:** Sync state transitions to `NoNetwork`. App remains responsive.
   No crash, no data loss.

## 5. Kill App Mid-Edit, Verify Edit Preserved

1. Open a note and start typing new content
2. Force-quit the app (swipe away from recents)
3. Reopen the app and navigate to the same note
4. **Expected:** Content is preserved (auto-saved via debounce before kill,
   or saved on lifecycle pause).

## 6. Server Returns 412 on PUT, Verify Client Handles

1. Note exists on server with hash `AAAA`
2. Manually edit the note on server (change hash to `BBBB`)
3. Client tries to PUT with `If-Match: AAAA`
4. **Expected:** Server returns 412 Precondition Failed. Client handles
   gracefully — logs the conflict, does not overwrite server version.

## 7. Large Note (Just Under 5 MB), Verify Upload

1. Create a note with ~4.9 MB of content
2. Save and sync
3. **Expected:** Note uploads successfully
4. Create a note with >5 MB of content
5. **Expected:** Server rejects with 413 Payload Too Large

## 8. 1000 Notes in Workspace, Verify Performance

1. Create 1000 notes across various folders
2. Open the app and navigate folders
3. Tap Sync Now
4. **Expected:** Manifest generation under 2 seconds. UI remains responsive.
   Folder listing loads without noticeable lag.

## 9. Folder Rename, Verify All Child Note Paths Update

1. Create folder `projects/` with notes `projects/a.md`, `projects/b.md`
2. Rename `projects/` to `work/`
3. **Expected:** Notes now at `work/a.md`, `work/b.md`. Manifest updated.
   Old paths no longer appear.

## 10. Restart Server in tmpfs Mode, Verify Snapshot Recovery

1. Start server with `NOTES_STORAGE_MODE=tmpfs`
2. Create several notes via the app
3. Wait for snapshot interval (default 300s) or manually trigger
4. Stop the server
5. Restart the server
6. **Expected:** All notes survive restart. No data loss.
