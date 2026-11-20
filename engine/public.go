package engine
import "time"

// BatchEntry is the manifest record used for two-stage atomic rollback.
type BatchEntry = rollbackEntry
// StageResult is the verified V3.8 engine output.
type StageResult = stagedResult

func Normal(root string, t time.Time) StageResult {return runNormalStage(root,t)}
func Face(root string, previous []BatchEntry, t time.Time) StageResult {return runFaceStage(root,previous,t)}
func Preview(root string,t time.Time) StageResult {return previewStage(root,t)}
func Rollback(root string,t time.Time) StageResult {return rollbackStage(root,t)}
func ValidateRoot(root string)(string,error){return validateRoot(root)}
