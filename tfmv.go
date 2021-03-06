package main

import (
	"fmt"
	"log"
	"os"
	"reflect"

	tfmt "github.com/hashicorp/terraform/command/format"
	"github.com/hashicorp/terraform/terraform"
	"flag"
)

func getPlan(file string) (fmtPlan *tfmt.Plan, err error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = f.Close()
	}()

	// Terraform has two Plan types, for some reason. `terraform.Plan` doesn't include the module in the address, so only use it for reading from the plan file, then convert it to the other one.
	plan, err := terraform.ReadPlan(f)
	if err != nil {
		return nil, err
	}
	fmtPlan = tfmt.NewPlan(plan)
	return fmtPlan, nil
}

func getChangesByType(plan *tfmt.Plan) (ChangesByType, error) {
	changesByType := ChangesByType{}

	// inspired by
	// https://github.com/palantir/tfjson/blob/master/tfjson.go
	for _, resource := range plan.Resources {
		changesByType.Add(*resource)
	}

	return changesByType, nil
}

func checkIfObjectsMatch(name string, creation, deletion interface{}) error {
	if reflect.DeepEqual(creation, deletion) {
		err := fmt.Errorf(name+" match, which they shouldn't:\ncreation: %+v\ndeletion:%+v\n", creation, deletion)
		return err
	}
	return nil
}

func getMoveStatements(plan *tfmt.Plan) ([]string, error) {
	moves := []string{}

	changesByType, err := getChangesByType(plan)
	if err != nil {
		return moves, err
	}

	for _, changes := range changesByType {
		for i, creation := range changes.Created {
			// stop if we're out of matches
			if i == len(changes.Destroyed) {
				break
			}
			deletion := changes.Destroyed[i]

			// sanity checks
			if err := checkIfObjectsMatch("Addrs", creation.Addr, deletion.Addr); err != nil {
				return moves, err
			}
			if err := checkIfObjectsMatch("InstanceDiffs", creation, deletion); err != nil {
				return moves, err
			}

			moves = append(moves, "terraform state mv "+deletion.Addr.String()+" "+creation.Addr.String())
		}
	}

	return moves, nil
}

func main() {
	planfile := flag.String("plan-file", "tfplan", "name of the plan-file")
	flag.Parse()

	plan, err := getPlan(*planfile)
	if err != nil {
		log.Fatalln(err)
	}

	moves, err := getMoveStatements(plan)
	if err != nil {
		log.Fatalln(err)
	}
	for _, move := range moves {
		fmt.Println(move)
	}
}
