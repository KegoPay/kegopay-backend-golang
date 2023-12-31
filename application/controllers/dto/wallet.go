package dto

type SendPaymentDTO struct {
	Pin         			string       	 `json:"pin"`
	FullName         		*string      	 `json:"fullName"`
	Amount      			uint64      	 `json:"amount"`
	DestinationCountryCode  string 		 	 `json:"destinationCountryCode"`
	BankCode				string 		 	 `json:"bankCode"`
	BranchCode				*string 		 `json:"branchCode"`
	AccountNumber 			string 			 `json:"accountNumber"`
	Description 			*string 		 `json:"description"`
	IPAddress 				string 			 `json:"ipAddress"`
}

type NameVerificationDTO struct {
	AccountNumber  string       `bson:"accountNumber" json:"accountNumber"`
	BankName       string       `bson:"bankName" json:"bankName"`
}

type SetPaymentTagDTO struct {
	Tag  string       `bson:"tag" json:"tag" validate:"required,alphanum,string_min_length_3"`
}