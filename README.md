<div align="center" alt="Warrant">
    <a href="https://warrant.dev/?utm_source=awesome-authz" target="_blank">
        <img src="https://warrant.dev/images/logo-primary-wide.png" width="300">
    </a>
    </br>
    </br>
</div>

# Warrant Edge Agent

[![Slack](https://img.shields.io/badge/slack-join-brightgreen)](https://join.slack.com/t/warrantcommunity/shared_invite/zt-12g84updv-5l1pktJf2bI5WIKN4_~f4w)

## What is the Warrant Edge Agent?

The Warrant Edge Agent is a lightweight service that can process Warrant access check requests. It can be deployed in any cloud environment to minimize the latency of access check requests from services using Warrant to enforce authorization. The Edge Agent serves access check requests from a local cache and connects to `stream.warrant.dev` to receive updates as access rules are modified in order to keep it's cache up-to-date.

## Should I use the Warrant Edge Agent?

Teams with strict latency and reliability requirements can deploy the Warrant Edge Agent in their own infrastructure for improved latency and availability.

## Getting Started

To get started with the Warrant Edge Agent, refer to our documentation on [Setting up the Edge Agent](https://docs.warrant.dev/quickstart/edge-agent).

## About Warrant

[Warrant](https://warrant.dev) provides APIs and infrastructure for implementing authorization and access control. Check out our [docs](https://docs.warrant.dev) to learn more.
