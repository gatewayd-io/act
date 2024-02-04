# Act

This repository is a PoC for the Signal, Action and Policy System that is proposed in the [gatewayd-io/proposals#5](https://github.com/gatewayd-io/proposals/issues/5) and (will be) implemented in the [gatewayd-io/gatewayd#422](https://github.com/gatewayd-io/gatewayd/issues/422).

The main goal of this repository is to provide a simple and easy to understand implementation of the Signal, Action and Policy System. This implementation is based on what was discussed in the proposal and it is not a final version of the system.

The Signal, Action and Policy System is a system that allows the user to define a set of rules that will be executed when a signal is received. The system is composed by four main components:

- **Signal**: A signal is a message that is sent to the system by the plugins. The signal is used to trigger the execution of the actions that are associated with the signal and the policy. The plugin can send multiple signals from a given hook function.
- **Policy**: A policy is a set of rules that will be used to define the behavior of the system when a signal is received. The policy is composed by a set of rules that will be executed in a specific order. The rules can be defined by the user and they can be used to define the behavior of the system when a signal is received.
- **Action**: An action is a function that will be executed when a signal is received. Actions are either sync or async.
- **Registry**: The registry is a component that is used to store the signals, actions and policies that are defined by the user. The registry is used to define the behavior of the system when a signal is received and run the actions that are associated with the signal and the policy.

Signals trigger actions and policies control those actions. For example, the `terminate` signal is returned by the plugin's `OnTrafficFromClient` hook function. The termination policy controls whether to run the action or not.
