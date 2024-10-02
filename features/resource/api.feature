Feature: API
#    Background:
#      Given a kdeps container with "GET, POST" endpoint "json" API and "/resource1, /resource2"

#    Scenario: GET request points to action
#      When I GET request to "/resource1?params1=1&params2=2" with data "hello" and header name "hello" that maps to "foo"
#      Then I should see a "request.pkl" in the "/agent/actions/api/" folder
#      And I should see action "GET", url "/resource1", data "hello", headers "hello,world" with values "foo,bar" and params "params1,params2" that maps to "1,2"
#      And I should see a blank standard template "response.pkl" in the "/agent/api" folder
#      When I fill in the "response.pkl" with success "true", response data "hello"
#      Then it should respond "hello" in "json"
