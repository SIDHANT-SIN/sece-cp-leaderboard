package handles

import (
//	"fmt"
	"net/http"

	// "leaderboard/src/repository"
	// "leaderboard/src/storage"
	// "leaderboard/src/utils"

	"github.com/gin-gonic/gin"
)

// CreateICPCProblem handles creating an ICPC problem, parsing inputs/outputs,
// uploading files to Azure Blob Storage, and saving URLs to database.
func CreateICPCProblem(c *gin.Context) {
	// --- AUTH CHECK ---
	cookie, err := c.Cookie("maintainer_logged_in")
	if err != nil || cookie != "true" {
		c.Redirect(http.StatusSeeOther, "/maintainer")
		return
	}

	// // --- 1. GATHER FORM DATA ---
	// title := c.PostForm("title")
	// statement := c.PostForm("statement")
	// inputDesc := c.PostForm("input_desc")
	// outputDesc := c.PostForm("output_desc")
	// constraints := c.PostForm("constraints")
	// sampleInput := c.PostForm("sample_input")
	// sampleOutput := c.PostForm("sample_output")
	// explanation := c.PostForm("explanation")

	// timeLimit := c.PostForm("time_limit")
	// memoryLimit := c.PostForm("memory_limit")

	// if timeLimit == "" {
	// 	timeLimit = "1"
	// }
	// if memoryLimit == "" {
	// 	memoryLimit = "256"
	// }

	// // --- 2. PARSE TESTCASES (Inputs) ---
	// testcaseJSON := c.PostForm("testcases")
	// testcases, err := utils.ParseTestCases(testcaseJSON)
	// if err != nil {
	// 	c.String(http.StatusBadRequest, "Invalid testcases JSON")
	// 	return
	// }

	// // --- 3. PARSE SOLUTIONS (Outputs) ---
	// solutionJSON := c.PostForm("solution_code")
	// solutions, err := utils.ParseSolutions(solutionJSON)
	// if err != nil {
	// 	c.String(http.StatusBadRequest, "Invalid solution JSON")
	// 	return
	// }

	// // Safety check: Make sure no one messed up the form submission
	// if len(testcases) != len(solutions) {
	// 	c.String(http.StatusBadRequest, "Mismatch: Number of inputs does not match number of outputs")
	// 	return
	// }

	// // --- 4. INSERT PROBLEM INTO DB ---
	// problemID, err := repository.InsertProblem(
	// 	title, statement,
	// 	timeLimit, memoryLimit,
	// 	inputDesc, outputDesc,
	// 	constraints,
	// 	sampleInput, sampleOutput,
	// 	explanation,
	// )
	// if err != nil {
	// 	c.String(http.StatusInternalServerError, "DB insert failed: %v", err)
	// 	return
	// }

	// // --- 5. PROCESS TESTCASES ---
	// for i := range testcases {
	// 	// Upload Input File to Azure
	// 	inputPath := fmt.Sprintf("problems/%d/tc_%d/input.txt", problemID, i)
	// 	inputURL, err := storage.UploadFile(inputPath, []byte(testcases[i].Input))
	// 	if err != nil {
	// 		c.String(http.StatusInternalServerError, fmt.Sprintf("Azure upload failed for input %d", i))
	// 		return
	// 	}

	// 	// Upload Output File to Azure
	// 	outputPath := fmt.Sprintf("problems/%d/tc_%d/output.txt", problemID, i)
	// 	outputURL, err := storage.UploadFile(outputPath, []byte(solutions[i].Output))
	// 	if err != nil {
	// 		c.String(http.StatusInternalServerError, fmt.Sprintf("Azure upload failed for output %d", i))
	// 		return
	// 	}

	// 	// Save both URLs to the database
	// 	err = repository.InsertTestcase(problemID, inputURL, outputURL)
	// 	if err != nil {
	// 		c.String(http.StatusInternalServerError, "DB testcase insert failed: %v", err)
	// 		return
	// 	}
	// }

	// --- 6. REDIRECT ON SUCCESS ---
	c.Redirect(http.StatusSeeOther, "/maintainer/icpc_pyq")
}