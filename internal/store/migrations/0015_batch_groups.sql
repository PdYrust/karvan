-- A rule of batches serves its old panel: every active batch with the
-- rule's old panel and destination, from every run, now and later. The
-- batches it was saved with no longer narrow it, so the table that held
-- them goes. Every existing rule of batches widens to all the active
-- batches of its old panel. Manual rules, batches, ledger rows and tokens
-- do not change; a token still names one ledger row, so one user.
DROP TABLE legacy_site_batches;
