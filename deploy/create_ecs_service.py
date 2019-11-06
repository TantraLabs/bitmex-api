import json
import os

algo_name = "swings-testnet"
aws_account_id = "911053277385"

with open('ecs_task_template.json', 'r') as f:
    ecs_task_template = json.load(f)

with open('ecs_service_template.json', 'r') as f:
    ecs_service_template = json.load(f)

    
def find_and_replace(data, find, replace):
    for key in data:
        if type(data[key]) is str:
            if find in data[key]:
                data[key] = data[key].replace(find, replace)
        elif type(data[key]) is dict:
            data[key] = find_and_replace(data[key], find, replace)
        elif type(data[key]) is list:
            for i in range(len(data[key])):
                if type(data[key][i]) is dict:
                    data[key][i] = find_and_replace(data[key][i], find, replace)

    return data

create_task_json = find_and_replace(ecs_task_template, "ALGO_NAME", algo_name)
create_task_json = json.dumps(find_and_replace(create_task_json, "AWS_ACCOUNT_ID", aws_account_id))

create_service_json = find_and_replace(ecs_service_template, "ALGO_NAME", algo_name)
create_service_json = json.dumps(find_and_replace(create_service_json, "AWS_ACCOUNT_ID", aws_account_id))


os.system("aws ecs register-task-definition --cli-input-json '" + create_task_json + "'")
os.system("aws ecs create-service --cluster MM-cluster --launch-type FARGATE --cli-input-json '" + create_service_json + "'")
# print(create_task_json)