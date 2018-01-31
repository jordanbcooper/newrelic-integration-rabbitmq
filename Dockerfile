FROM rabbitmq:3.7-management

# Ensure applications know there is no xterm
ENV DEBIAN_FRONTEND noninteractive

EXPOSE 15672

# Update the system and install dependencies
RUN apt-get update && apt-get install -y curl apt-transport-https apt-utils

# Get the newrelic gpg key
RUN curl https://download.newrelic.com/infrastructure_agent/gpg/newrelic-infra.gpg 2>/dev/null | apt-key add - 2>/dev/null

# Add the ppa to sources list
RUN printf "deb [arch=amd64] https://download.newrelic.com/infrastructure_agent/linux/apt trusty main" >> /etc/apt/sources.list.d/newrelic-infra.list

# Update package lists and install newrelic
RUN apt-get update && apt-get install newrelic-infra -y

# Add project files
COPY ./bin/rabbitmq_integration /var/db/newrelic-infra/custom-integrations/bin/rabbitmq_integration
COPY ./config/rabbitmq_integration-config.yml /etc/newrelic-infra/integrations.d/rabbitmq_integration-config.yml
COPY integration /var/db/newrelic-infra/custom-integrations/rabbitmq_integration-definition.yml

# Run rabbitmq service and newrelic-infra
CMD rabbitmq-server start -detached && newrelic-infra start

