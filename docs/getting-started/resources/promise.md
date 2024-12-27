---
outline: deep
---

# Promise Operator



Resource Promise ​
Notice that each resource function is enclosed within "@()". This follows the Kdeps convention, which ensures the resource is executed at a later stage. For more details on this convention, refer to the documentation on the Kdeps Promise directive.

When invoking a resource function, always wrap it in "@()" along with double quotes, as in "@(llm.response("chatResource"))". Depending on the output of this promise, you may sometimes needed to escape it.

For example:
