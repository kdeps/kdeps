package config

func primaryCloudProvider() cloudProvider {
	return cloudProvidersList[0]
}

func secondaryCloudProvider() cloudProvider {
	return cloudProvidersList[1]
}
