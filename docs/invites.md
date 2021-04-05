# Invites

This implementation of Rooms 2.0 is compliant with the [Rooms 2.0 specification](https://github.com/ssb-ngi-pointer/rooms2), but we add a few additional features and pages in order to improve user experience when their SSB app does not support [SSB URIs](https://github.com/ssb-ngi-pointer/ssb-uri-spec).

A summary can be seen in the following chart:

![Chart](./invites-chart.png)

When the browser and operating system detects no support for opening SSB URIs, we redirect to a fallback page which presents the user with two broad options: (1) install an SSB app that supports SSB URIs, (2) link to another page where the user can manually input the user's SSB ID in a form.
