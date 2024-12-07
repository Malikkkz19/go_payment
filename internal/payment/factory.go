package payment

import "fmt"

// ProviderType определяет тип платежного провайдера
type ProviderType string

const (
	ProviderStripe ProviderType = "stripe"
	ProviderPayPal ProviderType = "paypal"
)

// ProviderFactory создает экземпляры платежных провайдеров
type ProviderFactory struct {
	providers map[ProviderType]Provider
}

// NewProviderFactory создает новую фабрику провайдеров
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{
		providers: make(map[ProviderType]Provider),
	}
}

// RegisterProvider регистрирует провайдера в фабрике
func (f *ProviderFactory) RegisterProvider(providerType ProviderType, provider Provider) {
	f.providers[providerType] = provider
}

// CreateProvider создает и инициализирует провайдера заданного типа
func (f *ProviderFactory) CreateProvider(providerType ProviderType, config map[string]string) (Provider, error) {
	provider, exists := f.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("unsupported payment provider: %s", providerType)
	}

	if err := provider.Initialize(config); err != nil {
		return nil, fmt.Errorf("failed to initialize provider %s: %w", providerType, err)
	}

	return provider, nil
}

// GetSupportedProviders возвращает список поддерживаемых провайдеров
func (f *ProviderFactory) GetSupportedProviders() []ProviderType {
	providers := make([]ProviderType, 0, len(f.providers))
	for provider := range f.providers {
		providers = append(providers, provider)
	}
	return providers
}
