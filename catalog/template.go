package catalog

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"k8s.io/kubernetes/pkg/apis/extensions"
)

type JobType string

const (
	JobTypeOnCreateInstance JobType = "onCreateInstance"
	JobTypeOnDeleteInstance JobType = "onDeleteInstance"
	JobTypeOnBindInstance   JobType = "onBindInstance"
	JobTypeOnUnbindInstance JobType = "onUnbindInstance"
)

type Template struct {
	Id    string              `json:"id"`
	Body  KubernetesComponent `json:"body"`
	Hooks []*JobHook          `json:"hooks"`
}

type TemplateMetadata struct {
	Id                  string `json:"id"`
	TemplateDirName     string `json:"templateDirName"`
	TemplatePlanDirName string `json:"templatePlanDirName"`
}

type JobHook struct {
	Type JobType        `json:"type"`
	Job  extensions.Job `json:"job"`
}

var TEMPLATES map[string]*TemplateMetadata
var TemplatesPath string = "./catalogData/"
var CustomTemplatesDir string = TemplatesPath + "custom/"

func GetTemplateMetadataById(id string) *TemplateMetadata {
	if TEMPLATES != nil {
		return TEMPLATES[id]
	} else {
		return nil
	}
}

func GetAvailableTemplates() map[string]*TemplateMetadata {
	if TEMPLATES != nil {
		LoadAvailableTemplates()
	}
	return TEMPLATES
}

func LoadAvailableTemplates() {
	TEMPLATES = make(map[string]*TemplateMetadata)
	logger.Debug("GetAvailableTemplates - need to parse catalog/ directory.")
	template_file_info, err := ioutil.ReadDir(TemplatesPath)
	if err != nil {
		logger.Panic(err)
	}
	for _, templateDir := range template_file_info {
		loadTemplateMetadata(templateDir)
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
	for i, rc := range template.Body.Deployments {
		err := save_k8s_file_in_dir(templateDir, fmt.Sprintf("deployment_%d.json", i), rc)
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
	for i, job := range template.Hooks {
		err := save_k8s_file_in_dir(templateDir, fmt.Sprintf("job_%d.json", i), job)
		if err != nil {
			return err
		}
	}

	plan := PlanMetadata{Id: template.Id}
	err := save_k8s_file_in_dir(templatePlanDir, "plan.json", plan)
	if err != nil {
		return err
	}

	LoadAvailableTemplates()
	return nil
}

func RemoveAndUnregisterCustomTemplate(templateId string) error {
	templateDir := CustomTemplatesDir + templateId
	err := os.RemoveAll(templateDir)
	if err != nil {
		return err
	}

	LoadAvailableTemplates()
	return nil
}

func GetParsedTemplate(templateMetadata *TemplateMetadata, catalogPath, instanceId, orgId, spaceId string) (Template, error) {
	result := Template{Id: templateMetadata.Id}
	component, err := GetParsedKubernetesComponentByTemplate(catalogPath, instanceId, orgId, spaceId, templateMetadata)
	if err != nil {
		return result, err
	}

	jobsHooksRaw, err := GetJobHooks(catalogPath, templateMetadata)
	if err != nil {
		return result, err
	}

	jobHooks, err := GetParsedJobHooks(jobsHooksRaw, instanceId, templateMetadata.Id, templateMetadata.Id, orgId, spaceId)
	if err != nil {
		return result, err
	}

	result.Body = *component
	result.Hooks = jobHooks
	return result, nil
}

func GetRawTemplate(templateMetadata *TemplateMetadata, catalogPath string) (Template, error) {
	result := Template{Id: templateMetadata.Id}
	blueprint, err := GetKubernetesBlueprint(catalogPath, templateMetadata.TemplateDirName, templateMetadata.TemplatePlanDirName, templateMetadata.Id)
	if err != nil {
		return result, err
	}

	component, err := CreateKubernetesComponentFromBlueprint(blueprint, true)
	if err != nil {
		return result, err
	}

	jobsHooksRaw, err := GetJobHooks(catalogPath, templateMetadata)
	if err != nil {
		return result, err
	}

	jobHooks, err := unmarshallJobs(jobsHooksRaw)
	if err != nil {
		return result, err
	}

	result.Body = *component
	result.Hooks = jobHooks
	return result, nil
}

func GetParsedJobHooks(jobs []string, instanceId, svcMetaId, planMetaId, org, space string) ([]*JobHook, error) {
	parsedJobs := []string{}
	for i, job := range jobs {
		parsedJobs = append(parsedJobs, adjust_params(job, org, space, instanceId, svcMetaId, planMetaId, i))
	}
	return unmarshallJobs(parsedJobs)
}

func unmarshallJobs(jobs []string) ([]*JobHook, error) {
	result := []*JobHook{}
	for _, job := range jobs {
		jobHook := &JobHook{}
		err := json.Unmarshal([]byte(job), jobHook)
		if err != nil {
			logger.Error("Unmarshalling JobHook error:", err)
			return result, err
		}
		result = append(result, jobHook)
	}
	return result, nil
}

func GetJobHooks(catalogPath string, temp *TemplateMetadata) ([]string, error) {
	_, _, k8sPlanPath := GetCatalogFilesPath(catalogPath, temp.TemplateDirName, temp.TemplatePlanDirName)
	return read_k8s_json_files_with_prefix_from_dir(k8sPlanPath, "job")
}
