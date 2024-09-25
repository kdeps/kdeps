Feature: Kdeps CLI application
#   Background:
#     Given ".kdeps.pkl" does not exists in both home and current directory
#     And when kdeps is executed
#     Then the configuration will be generated in the home directory
#     And the default system folder will be created in the home directory

#   Scenario: Ability to bootstrap the Docker environment
#     Given a kdeps docker image with kdeps entrypoint
#     When the docker image container is started
#     Then kdeps will check the presence of the "/.dockerenv" file
#     And it will install the models defined in the ".kdeps.pkl" configuration if found
