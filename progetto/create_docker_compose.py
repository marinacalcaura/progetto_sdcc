import sys
import ruamel.yaml
import json

def read_config(file_path):
    try:
        with open(file_path, 'r') as config_file:
            config_data = json.load(config_file)
            return config_data
    except FileNotFoundError:
        print(f"File di configurazione '{file_path}' non trovato.")
        return None
    except json.JSONDecodeError:
        print(f"Errore nel parsing del file di configurazione '{file_path}'. Assicurati che sia in formato JSON valido.")
        return None



def create_docker_compose(num_nodi, num_bit):

    # Definizione dei servizi
    services = {
        's_registry': {
            'build': {
                'context':'./',
                'dockerfile':'./s_registry/Dockerfile'
            },
            'ports': [ruamel.yaml.scalarstring.DoubleQuotedScalarString('8000:8000')],
            'networks': ['chord-network'],
        }
    }

    for i in range(1, num_nodi + 1):
        id = str(8000 + i)
        service_name = f'nodo{id}'
        services[service_name] = {
            'build': {
                'context':'./',
                'dockerfile':'./node/Dockerfile'
            },
            'depends_on': ['s_registry'],
            'environment': [f"NODE_PORT={id}", f"NODE_HOST=node{i}-container", f"BIT={num_bit}"],
            'ports': [ruamel.yaml.scalarstring.DoubleQuotedScalarString(f'{id}:{id}')],
            'networks': ['chord-network'],
        }

    # Configurazione finale del docker-compose.yml
    docker_compose_config = ruamel.yaml.comments.CommentedMap({
        'version': '2.21',
        'services': services,
        'networks': {
            'chord-network': {
                'driver': "bridge"

            }
        } 
    })

    with open('docker-compose.yml', 'w') as compose_file:
        yaml = ruamel.yaml.YAML()
        yaml.indent(sequence=4, offset=2)
        yaml.dump(docker_compose_config, compose_file)
        


def create_docker_compose_from_config(config_file_path):
    config_data = read_config(config_file_path)

    if config_data:
        num_nodi = config_data.get("num_nodi")
        num_bit = config_data.get("num_bit")
        
        if num_nodi is not None and num_bit is not None:
            create_docker_compose(num_nodi, num_bit)
        else:
            print("Il file di configurazione non contiene informazioni complete.")

if __name__ == "__main__":
    config_file_path = "C:/Users/Marina/Desktop/sdcc2/config.json"
    create_docker_compose_from_config(config_file_path)

