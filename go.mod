module github.com/kaizjen/ogo-disconnected // just so it doesn't connect to github somehow

go 1.23

// I was fighting a random error in the package manager for like 45 minutes, the solution to which
// was to rename the folder "pkg/ogo" to "pkg/lib" - i like it better this way anyways.
// But i can't reproduce it!!! Which is super weird because everything in this repo was crashing
// and burning and i thought it was some stupid design, but it was just a one-time bug??