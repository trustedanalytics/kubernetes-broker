package catalog

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
)

type Template struct {
	Id   string              `json:"id"`
	Body KubernetesComponent `json:"body"`
}

type TemplateMetadata struct {
	Id                  string
	TemplateDirName     string
	TemplatePlanDirName string
}

var TEMPLATES map[string]*TemplateMetadata
var TemplatesPath string = "./catalogData/"
var CustomTemplatesDir string = TemplatesPath + "custom/"
var template_mutex sync.RWMutex

func GetTemplateMetadataById(id string) *TemplateMetadata {
	if TEMPLATES != nil {
		return TEMPLATES[id]
	} else {
		return nil
	}
}

func GetAvailableTemplates() map[string]*TemplateMetadata {
	if TEMPLATES != nil {
		return TEMPLATES
	} else {
		TEMPLATES = make(map[string]*TemplateMetadata)
		logger.Debug("GetAvailableTemplates - need to parse catalog/ directory.")
		template_file_info, err := ioutil.ReadDir(TemplatesPath)
		if err != nil {
			logger.Panic(err)
		}
		for _, templateDir := range template_file_info {
			loadTemplateMetadata(templateDir)
		}
		return TEMPLATES
	}
}

func loadTemplateMetadata(templateDir os.FileInfo) {
	if templateDir.IsDir() {
		templateDirPath := TemplatesPath + templateDir.Name()
		logger.Debug(" => ", templateDir.Name(), templateDirPath)

		plans_file_info, err := ioutil.ReadDir(templateDirPath)
		if err != nil {
			logger.Panic(err)
		}
		for _, plandir := range plans_file_info {
			loadPlans(plandir, templateDirPath, templateDir.Name())
		}
	}
}

func loadPlans(plandir os.FileInfo, templateDirPath, templateDirName string) {
	planDirPath := templateDirPath + "/" + plandir.Name()

	if plandir.IsDir() {
		logger.Debug(" ====> ", plandir.Name(), planDirPath)
		plans_content_file_info, err := ioutil.ReadDir(planDirPath)
		if err != nil {
			logger.Panic(err)
		}

		for _, plan_details := range plans_content_file_info {
			loadPlan(plan_details, planDirPath, plandir.Name(), templateDirName)
		}
	} else {
		logger.Debug("Skipping file: ", planDirPath)
	}
}

func loadPlan(plan_details os.FileInfo, planDirPath, planDirName, templateDirName string) {
	var plan_meta PlanMetadata
	plan_details_dir_full_name := planDirPath + "/" + plan_details.Name()
	if plan_details.IsDir() {
		logger.Debug("Skipping directory:", plan_details_dir_full_name)
	} else if plan_details.Name() == "plan.json" {
		plan_metadata_file_content, err := ioutil.ReadFile(plan_details_dir_full_name)
		if err != nil {
			logger.Fatal("Error reading file: ", plan_details_dir_full_name, err)
		}
		b := []byte(plan_metadata_file_content)
		err = json.Unmarshal(b, &plan_meta)
		if err != nil {
			logger.Fatal("Error parsing json from file: ", plan_details_dir_full_name, err)
		}

		TEMPLATES[plan_meta.Id] = &TemplateMetadata{
			Id:                  plan_meta.Id,
			TemplateDirName:     templateDirName,
			TemplatePlanDirName: planDirName,
		}
	} else {
		logger.Debug(" -----------> ", plan_details.Name(), plan_details_dir_full_name)
	}
}

func AddAndRegisterCustomTemplate(template Template) error {
	templateDir := CustomTemplatesDir + template.Id + "/k8s"
	templatePlanDir := CustomTemplatesDir + template.Id

	for i, pvc := range template.Body.PersistentVolumeClaims {
		err := save_k8s_file_in_dir(templateDir, fmt.Sprintf("persistentvolumeclaim_%d.json", i), pvc)
		if err != nil {
			return err
		}
	}
	for i, rc := range template.Body.ReplicationControllers {
		err := save_k8s_file_in_dir(templateDir, fmt.Sprintf("replicationcontroller_%d.json", i), rc)
		if err != nil {
			return err
		}
	}
	for i, svc := range template.Body.Services {
		err := save_k8s_file_in_dir(templateDir, fmt.Sprintf("service_%d.json", i), svc)
		if err != nil {
			return err
		}
	}
	for i, svcAccount := range template.Body.ServiceAccounts {
		err := save_k8s_file_in_dir(templateDir, fmt.Sprintf("account_%d.json", i), svcAccount)
		if err != nil {
			return err
		}
	}
	for i, secret := range template.Body.Secrets {
		err := save_k8s_file_in_dir(templateDir, fmt.Sprintf("secret_%d.json", i), secret)
		if err != nil {
			return err
		}
	}

	plan := PlanMetadata{Id: template.Id}
	err := save_k8s_file_in_dir(templatePlanDir, "plan.json", plan)
	if err != nil {
		return err
	}
	registerTemplateInCatalog(&TemplateMetadata{
		Id:                  template.Id,
		TemplatePlanDirName: templatePlanDir,
		TemplateDirName:     templateDir,
	})
	return nil
}

func registerTemplateInCatalog(template *TemplateMetadata) {
	template_mutex.Lock()
	TEMPLATES[template.Id] = template
	template_mutex.Unlock()
	logger.Info(fmt.Sprintf("Template %s registred in catalog!", template.Id))
}
